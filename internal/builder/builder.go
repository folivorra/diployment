package builder

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/pkg/crypto/aesgcm"
	"github.com/folivorra/diployment/pkg/retry"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/image"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/labstack/gommon/bytes"
	"github.com/minio/minio-go/v7"
	"github.com/moby/go-archive"
	"github.com/moby/moby/client"
)

const (
	BucketArtifacts = "artifacts"
	BucketLogs      = "logs"

	ContentTypeXTar      = "application/x-tar"
	ContentTypeTextPlain = "text/plain"

	TmpDirName        = "builds"
	TmpLogNamePattern = "%s.log"

	UnknownSize = -1
)

// LogPublisher стримит строки лога в NATS.
type LogPublisher interface {
	PublishJobsLogLine(ctx context.Context, logLine model.JobLogLine) error
}

type builder struct {
	docker       *client.Client
	s3           *minio.Client
	l            LogPublisher
	key          []byte
	buildTimeout time.Duration
}

// NewBuilder создаёт билдер с docker-клиентом и зависимостями.
func NewBuilder(d *client.Client, s3 *minio.Client, l LogPublisher, masterKey []byte, buildTimeout time.Duration) *builder {
	return &builder{docker: d, s3: s3, l: l, key: masterKey, buildTimeout: buildTimeout}
}

// Build выполняет полный воркфлоу: подтягивает репозиторий, собирает проект и сохраняет артефактов.
func (b *builder) Build(ctx context.Context, event model.BuildDispatchEvent) (string, error) {
	token, err := aesgcm.Decrypt(event.EncryptedToken, b.key, aesgcm.GitHubToken)
	if err != nil {
		return "", fmt.Errorf("decrypt clone token: %w", err)
	}

	// создаём временную директорию для исходников, удаляем после завершения
	tmpDir, err := os.MkdirTemp("", TmpDirName)
	if err != nil {
		return "", fmt.Errorf("create tmp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cloneOpts := &git.CloneOptions{
		URL:           event.CloneURL,
		ReferenceName: plumbing.NewBranchReferenceName(event.Branch),
		SingleBranch:  true,
		Depth:         1,
		Auth:          &githttp.BasicAuth{Username: "x-token", Password: string(token)},
	}

	// клонируем только нужную ветку без истории, ретрай на случай сетевых проблем
	err = retry.WithRetry(ctx, retry.DefaultAttempts, retry.DefaultWait, func() error {
		// чистим перед каждой попыткой на случай частичного клона
		if err := os.RemoveAll(tmpDir); err != nil {
			return err
		}
		if err := os.MkdirAll(tmpDir, 0o700); err != nil {
			return err
		}
		_, err := git.PlainCloneContext(ctx, tmpDir, false, cloneOpts)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("git clone: %w", err)
	}

	// создаём файл для накопления лога сборки: пишем на диск, а не в память, чтобы не словить OOM
	logKey := fmt.Sprintf(TmpLogNamePattern, event.JobID.String())
	logFile, err := os.CreateTemp(tmpDir, logKey)
	if err != nil {
		return "", fmt.Errorf("create tmp log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	// упаковываем директорию в tar, docker daemon принимает build context именно так
	tar, err := archive.TarWithOptions(tmpDir, &archive.TarOptions{})
	if err != nil {
		return "", fmt.Errorf("tar build context: %w", err)
	}
	defer func() { _ = tar.Close() }()

	// запускаем сборку образа, тег = job_id
	tags := []string{event.JobID.String()}
	buildCtx, buildCancel := context.WithTimeout(ctx, b.buildTimeout)
	defer buildCancel()
	bresp, err := b.docker.ImageBuild(buildCtx, tar, build.ImageBuildOptions{
		Tags:       tags,
		Dockerfile: "Dockerfile",
		Remove:     true,
	})
	if err != nil {
		return "", fmt.Errorf("image build: %w", err)
	}
	defer func() { _ = bresp.Body.Close() }()

	// читаем вывод сборки, без этого daemon не завершит сборку
	// каждая строка это JSON, проверяем на наличие ошибки сборки
	// лог пишем в файл и пушим в стрим
	scanner := bufio.NewScanner(bresp.Body)
	scanner.Buffer(make([]byte, 512*bytes.KiB), 512*bytes.KiB)
	writer := bufio.NewWriter(logFile)
	for scanner.Scan() {
		var msg struct {
			Stream string `json:"stream"`
			Error  string `json:"error"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &msg); err == nil && msg.Error != "" {
			return "", fmt.Errorf("docker build: %s", msg.Error)
		}

		if _, err = writer.WriteString(msg.Stream); err != nil {
			return "", fmt.Errorf("print log line into file: %w", err)
		}

		if err := b.l.PublishJobsLogLine(ctx, model.JobLogLine{
			JobID: event.JobID,
			Line:  msg.Stream,
			Phase: model.PhaseBuild,
		}); err != nil {
			slog.Warn("failed to pub log line",
				slog.String("job_id", event.JobID.String()),
				slog.Any("error", err),
			)
		}
	}
	if err := writer.Flush(); err != nil {
		return "", fmt.Errorf("flush last bytes: %w", err)
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read build output: %w", err)
	}

	// экспортируем образ и стримим в MinIO, ретрай на случай инфраструктурных проблем
	// ImageSave вызывается внутри ретрая, тк стрим нельзя перечитать повторно
	err = retry.WithRetry(ctx, retry.DefaultAttempts, retry.DefaultWait, func() error {
		imageReader, err := b.docker.ImageSave(ctx, tags)
		if err != nil {
			return err
		}
		defer func() { _ = imageReader.Close() }()

		_, err = b.s3.PutObject(ctx, BucketArtifacts, event.JobID.String()+".tar", imageReader, UnknownSize,
			minio.PutObjectOptions{ContentType: ContentTypeXTar},
		)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("upload artifact: %w", err)
	}

	// перематываем файл и загружаем лог в MinIO
	// некритично, если не вышло, logKey обнуляем, чтобы координатор не записал в базу невалидную ссылку
	if err = retry.WithRetry(ctx, retry.DefaultAttempts, retry.DefaultWait, func() error {
		if _, err := logFile.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("seek log file: %w", err)
		}
		fi, _ := logFile.Stat()
		_, err = b.s3.PutObject(ctx, BucketLogs, logKey, logFile, fi.Size(),
			minio.PutObjectOptions{ContentType: ContentTypeTextPlain},
		)
		return err
	}); err != nil {
		logKey = ""
		slog.Warn("failed to put log file into s3",
			slog.String("job_id", event.JobID.String()),
			slog.Any("error", err),
		)
	}

	// публикуем сентинель, SSE-клиенты по нему понимают, что лог завершён
	if err := b.l.PublishJobsLogLine(ctx, model.JobLogLine{
		JobID: event.JobID,
		Phase: model.PhaseBuild,
		Done:  true,
	}); err != nil {
		slog.Warn("failed to pub log line",
			slog.String("job_id", event.JobID.String()),
			slog.Any("error", err),
		)
	}

	// удаляем локальный образ из docker daemon чтобы не забивать диск
	// некритично: логируем и продолжаем если не получилось
	if _, err = b.docker.ImageRemove(ctx, tags[0], image.RemoveOptions{Force: true}); err != nil {
		slog.Warn("failed to remove docker image",
			slog.String("job_id", event.JobID.String()),
			slog.Any("error", err),
		)
	}

	return logKey, nil
}
