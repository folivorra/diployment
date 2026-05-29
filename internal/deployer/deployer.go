package deployer

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/folivorra/diployment/internal/model"
	miniorepo "github.com/folivorra/diployment/internal/repository/minio"
	"github.com/folivorra/diployment/pkg/crypto/aesgcm"
	"github.com/folivorra/diployment/pkg/retry"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	ContentTypeTextPlain = "text/plain"

	TmpDirName = "deploys"
)

// LogPublisher стримит строки лога в NATS.
type LogPublisher interface {
	PublishJobsLogLine(ctx context.Context, logLine model.JobLogLine) error
}

type deployer struct {
	s3             *minio.Client
	l              LogPublisher
	key            []byte
	deployTimeout  time.Duration
	sshDialTimeout time.Duration
}

// NewDeployer создаёт деплоер с MinIO-клиентом и зависимостями.
func NewDeployer(s3 *minio.Client, l LogPublisher, masterKey []byte, deployTimeout, sshDialTimeout time.Duration) *deployer {
	return &deployer{s3: s3, l: l, key: masterKey, deployTimeout: deployTimeout, sshDialTimeout: sshDialTimeout}
}

// Deploy выполняет деплой и всегда сохраняет лог:
// при успехе - вывод restart-команды;
// при ошибке - то, что успели написать, плюс саму ошибку последней строкой.
// Возвращает ключ лога в S3 и ошибку деплоя.
func (d *deployer) Deploy(ctx context.Context, event model.DeployDispatchEvent) (string, error) {
	var logBuf bytes.Buffer
	publishLine := func(line string) {
		logBuf.WriteString(line)
		if err := d.l.PublishJobsLogLine(ctx, model.JobLogLine{
			JobID: event.JobID,
			Line:  line,
			Phase: model.PhaseDeploy,
		}); err != nil {
			slog.Warn("failed to pub deploy log line",
				slog.String("job_id", event.JobID.String()),
				slog.Any("error", err),
			)
		}
	}

	deployErr := d.runDeploy(ctx, event, publishLine)
	if deployErr != nil {
		publishLine(fmt.Sprintf("error: %s\n", deployErr.Error()))
	}

	logKey := d.uploadLog(ctx, event.JobID, logBuf.Bytes())

	if err := d.l.PublishJobsLogLine(ctx, model.JobLogLine{
		JobID: event.JobID,
		Phase: model.PhaseDeploy,
		Done:  true,
	}); err != nil {
		slog.Warn("failed to pub deploy log sentinel",
			slog.String("job_id", event.JobID.String()),
			slog.Any("error", err),
		)
	}

	return logKey, deployErr
}

// runDeploy расшифровывает SSH-ключ, тянет артефакт, заливает на сервер
// и запускает restart-команду. Все промежуточные шаги пишутся через publishLine.
func (d *deployer) runDeploy(ctx context.Context, event model.DeployDispatchEvent, publishLine func(string)) error {
	deployCtx, cancel := context.WithTimeout(ctx, d.deployTimeout)
	defer cancel()

	rawKey, err := aesgcm.Decrypt(event.EncryptedSSHKey, d.key, aesgcm.DeploySSHKey)
	if err != nil {
		return fmt.Errorf("decrypt ssh key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(rawKey)
	if err != nil {
		return fmt.Errorf("parse ssh private key: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", TmpDirName)
	if err != nil {
		return fmt.Errorf("create tmp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	artifactKey := fmt.Sprintf("%s.tar", event.JobID.String())
	localTar, err := os.CreateTemp(tmpDir, artifactKey)
	if err != nil {
		return fmt.Errorf("create tmp artifact file: %w", err)
	}
	defer func() { _ = localTar.Close() }()

	publishLine(fmt.Sprintf("downloading artifact %s from object storage\n", artifactKey))
	if err := retry.WithRetry(deployCtx, retry.DefaultAttempts, retry.DefaultWait, func() error {
		obj, err := d.s3.GetObject(deployCtx, miniorepo.BucketArtifacts, artifactKey, minio.GetObjectOptions{})
		if err != nil {
			return err
		}
		defer func() { _ = obj.Close() }()
		if err := localTar.Truncate(0); err != nil {
			return err
		}
		if _, err := localTar.Seek(0, io.SeekStart); err != nil {
			return err
		}
		_, err = io.Copy(localTar, obj)
		return err
	}); err != nil {
		return fmt.Errorf("download artifact from s3: %w", err)
	}

	addr := net.JoinHostPort(event.SSHHost, fmt.Sprintf("%d", event.SSHPort))
	publishLine(fmt.Sprintf("connecting to %s as %s\n", addr, event.SSHUser))
	sshCfg := &ssh.ClientConfig{
		User:            event.SSHUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         d.sshDialTimeout,
	}
	sshClient, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return fmt.Errorf("ssh dial %s: %w", addr, err)
	}
	defer func() { _ = sshClient.Close() }()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("sftp client: %w", err)
	}
	defer func() { _ = sftpClient.Close() }()

	remoteTar := fmt.Sprintf("%s/%s.tar", event.Workdir, event.JobID.String())

	publishLine(fmt.Sprintf("uploading artifact to %s\n", remoteTar))

	if _, err := localTar.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek artifact file: %w", err)
	}
	remoteFile, err := sftpClient.Create(remoteTar)
	if err != nil {
		return fmt.Errorf("sftp create remote file: %w", err)
	}
	if _, err := io.Copy(remoteFile, localTar); err != nil {
		_ = remoteFile.Close()
		return fmt.Errorf("sftp upload artifact: %w", err)
	}
	_ = remoteFile.Close()

	publishLine(fmt.Sprintf("running restart command: %s\n", event.RestartCmd))
	restartErr := d.runRemoteCmd(sshClient, event, publishLine)

	publishLine("cleaning up remote artifact\n")
	cleanSession, _ := sshClient.NewSession()
	if cleanSession != nil {
		_, _ = cleanSession.CombinedOutput(fmt.Sprintf("rm -f %s", remoteTar))
		_ = cleanSession.Close()
	}

	if restartErr != nil {
		return fmt.Errorf("restart command failed: %w", restartErr)
	}

	return nil
}

// uploadLog грузит накопленный лог в MinIO. Возвращает ключ или "" если загрузка
// не удалась. В этом случае координатор не запишет ссылку, но live-лог пользователь всё равно увидит.
func (d *deployer) uploadLog(ctx context.Context, jobID uuid.UUID, logBytes []byte) string {
	key := fmt.Sprintf("%s-deploy.log", jobID.String())
	if err := retry.WithRetry(ctx, retry.DefaultAttempts, retry.DefaultWait, func() error {
		_, err := d.s3.PutObject(ctx, miniorepo.BucketLogs, key,
			bytes.NewReader(logBytes), int64(len(logBytes)),
			minio.PutObjectOptions{ContentType: ContentTypeTextPlain},
		)
		return err
	}); err != nil {
		slog.Warn("failed to upload deploy log to s3",
			slog.String("job_id", jobID.String()),
			slog.Any("error", err),
		)
		return ""
	}
	return key
}

// runRemoteCmd запускает event.RestartCmd на удалённом хосте, стримит вывод
// построчно через publishLine. Возвращает ошибку если exit code != 0.
func (d *deployer) runRemoteCmd(client *ssh.Client, event model.DeployDispatchEvent, publishLine func(string)) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("ssh new session: %w", err)
	}
	defer func() { _ = session.Close() }()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	cmd := fmt.Sprintf("cd %s && %s", event.Workdir, event.RestartCmd)
	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("start remote cmd: %w", err)
	}

	combined := io.MultiReader(stdout, stderr)
	scanner := bufio.NewScanner(combined)
	for scanner.Scan() {
		publishLine(scanner.Text() + "\n")
	}

	if err := session.Wait(); err != nil {
		return err
	}

	return nil
}
