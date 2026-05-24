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
	"github.com/folivorra/diployment/pkg/crypto/aesgcm"
	"github.com/folivorra/diployment/pkg/retry"

	"github.com/minio/minio-go/v7"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	BucketArtifacts = "artifacts"
	BucketLogs      = "logs"

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

// Deploy выполняет полный деплой: расшифровывает SSH-ключ, скачивает артефакт из MinIO,
// заливает на сервер через SFTP, запускает restart-команду и стримит её вывод.
func (d *deployer) Deploy(ctx context.Context, event model.DeployDispatchEvent) (string, error) {
	deployCtx, cancel := context.WithTimeout(ctx, d.deployTimeout)
	defer cancel()

	rawKey, err := aesgcm.Decrypt(event.EncryptedSSHKey, d.key, aesgcm.DeploySSHKey)
	if err != nil {
		return "", fmt.Errorf("decrypt ssh key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(rawKey)
	if err != nil {
		return "", fmt.Errorf("parse ssh private key: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", TmpDirName)
	if err != nil {
		return "", fmt.Errorf("create tmp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	artifactKey := fmt.Sprintf("%s.tar", event.JobID.String())
	localTar, err := os.CreateTemp(tmpDir, artifactKey)
	if err != nil {
		return "", fmt.Errorf("create tmp artifact file: %w", err)
	}
	defer func() { _ = localTar.Close() }()

	if err := retry.WithRetry(deployCtx, retry.DefaultAttempts, retry.DefaultWait, func() error {
		obj, err := d.s3.GetObject(deployCtx, BucketArtifacts, artifactKey, minio.GetObjectOptions{})
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
		return "", fmt.Errorf("download artifact from s3: %w", err)
	}

	addr := net.JoinHostPort(event.SSHHost, fmt.Sprintf("%d", event.SSHPort))
	sshCfg := &ssh.ClientConfig{
		User:            event.SSHUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         d.sshDialTimeout,
	}
	sshClient, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return "", fmt.Errorf("ssh dial %s: %w", addr, err)
	}
	defer func() { _ = sshClient.Close() }()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return "", fmt.Errorf("sftp client: %w", err)
	}
	defer func() { _ = sftpClient.Close() }()

	remoteTar := fmt.Sprintf("%s/%s.tar", event.Workdir, event.JobID.String())

	d.publishLog(ctx, event, fmt.Sprintf("uploading artifact to %s\n", remoteTar))

	if _, err := localTar.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("seek artifact file: %w", err)
	}
	remoteFile, err := sftpClient.Create(remoteTar)
	if err != nil {
		return "", fmt.Errorf("sftp create remote file: %w", err)
	}
	if _, err := io.Copy(remoteFile, localTar); err != nil {
		_ = remoteFile.Close()
		return "", fmt.Errorf("sftp upload artifact: %w", err)
	}
	_ = remoteFile.Close()

	d.publishLog(ctx, event, fmt.Sprintf("running restart command: %s\n", event.RestartCmd))

	logBuf, restartErr := d.runRemoteCmd(deployCtx, sshClient, event)

	d.publishLog(ctx, event, "cleaning up remote artifact\n")
	cleanSession, _ := sshClient.NewSession()
	if cleanSession != nil {
		_, _ = cleanSession.CombinedOutput(fmt.Sprintf("rm -f %s", remoteTar))
		_ = cleanSession.Close()
	}

	logKey := fmt.Sprintf("%s-deploy.log", event.JobID.String())
	logBytes := logBuf.Bytes()

	if uploadErr := retry.WithRetry(ctx, retry.DefaultAttempts, retry.DefaultWait, func() error {
		_, err := d.s3.PutObject(ctx, BucketLogs, logKey,
			bytes.NewReader(logBytes), int64(len(logBytes)),
			minio.PutObjectOptions{ContentType: ContentTypeTextPlain},
		)
		return err
	}); uploadErr != nil {
		logKey = ""
		slog.Warn("failed to upload deploy log to s3",
			slog.String("job_id", event.JobID.String()),
			slog.Any("error", uploadErr),
		)
	}

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

	if restartErr != nil {
		return logKey, fmt.Errorf("restart command failed: %w", restartErr)
	}

	return logKey, nil
}

// runRemoteCmd запускает event.RestartCmd на удалённом хосте, стримит вывод построчно
// в NATS и возвращает полный лог + ошибку если exit code != 0.
func (d *deployer) runRemoteCmd(ctx context.Context, client *ssh.Client, event model.DeployDispatchEvent) (*bytes.Buffer, error) {
	session, err := client.NewSession()
	if err != nil {
		return &bytes.Buffer{}, fmt.Errorf("ssh new session: %w", err)
	}
	defer func() { _ = session.Close() }()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return &bytes.Buffer{}, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return &bytes.Buffer{}, fmt.Errorf("stderr pipe: %w", err)
	}

	cmd := fmt.Sprintf("cd %s && %s", event.Workdir, event.RestartCmd)
	if err := session.Start(cmd); err != nil {
		return &bytes.Buffer{}, fmt.Errorf("start remote cmd: %w", err)
	}

	var logBuf bytes.Buffer
	combined := io.MultiReader(stdout, stderr)
	scanner := bufio.NewScanner(combined)
	for scanner.Scan() {
		line := scanner.Text() + "\n"
		logBuf.WriteString(line)
		d.publishLog(ctx, event, line)
	}

	if err := session.Wait(); err != nil {
		return &logBuf, err
	}
	return &logBuf, nil
}

func (d *deployer) publishLog(ctx context.Context, event model.DeployDispatchEvent, line string) {
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
