package worker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/pkg/crypto/aesgcm"
	"github.com/folivorra/diployment/pkg/retry"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/image"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/minio/minio-go/v7"
	"github.com/moby/go-archive"
	"github.com/moby/moby/client"
)

const (
	BucketArtifacts = "artifacts"
	ContentTypeXTar = "application/x-tar"
	TmpDirName      = "builds"
	UnknownSize     = -1
)

type builder struct {
	docker *client.Client
	s3     *minio.Client
	key    []byte
}

func NewBuilder(d *client.Client, s3 *minio.Client, masterKey []byte) *builder {
	return &builder{docker: d, s3: s3, key: masterKey}
}

// Build выполняет полный воркфлоу: подтягивает репозиторий, собирает проект и сохраняет артефактов.
func (b *builder) Build(ctx context.Context, job model.Job) error {
	token, err := aesgcm.Decrypt(job.EncryptedToken, b.key, []byte("USER-DATA"))
	if err != nil {
		return fmt.Errorf("decrypt clone token: %w", err)
	}

	// создаём временную директорию для исходников, удаляем после завершения
	tmpDir, err := os.MkdirTemp("", TmpDirName)
	if err != nil {
		return fmt.Errorf("create tmp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cloneOpts := &git.CloneOptions{
		URL:           job.CloneURL,
		ReferenceName: plumbing.NewBranchReferenceName(job.Branch),
		SingleBranch:  true,
		Depth:         1,
		Auth:          &githttp.BasicAuth{Username: "x-token", Password: string(token)},
	}

	// клонируем только нужную ветку без истории, ретрай на случай сетевых проблем
	err = retry.WithRetry(retry.DefaultAttempts, retry.DefaultWait, func() error {
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
		return fmt.Errorf("git clone: %w", err)
	}

	// упаковываем директорию в tar, docker daemon принимает build context именно так
	tar, err := archive.TarWithOptions(tmpDir, &archive.TarOptions{})
	if err != nil {
		return fmt.Errorf("tar build context: %w", err)
	}
	defer func() { _ = tar.Close() }()

	// запускаем сборку образа, тег = job_id
	tags := []string{job.ID.String()}
	bresp, err := b.docker.ImageBuild(ctx, tar, build.ImageBuildOptions{
		Tags:       tags,
		Dockerfile: "Dockerfile",
		Remove:     true,
	})
	if err != nil {
		return fmt.Errorf("image build: %w", err)
	}
	defer func() { _ = bresp.Body.Close() }()

	// читаем вывод сборки, без этого daemon не завершит сборку
	// каждая строка это JSON, проверяем на наличие ошибки сборки
	scanner := bufio.NewScanner(bresp.Body)
	for scanner.Scan() {
		var msg struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &msg); err == nil && msg.Error != "" {
			return fmt.Errorf("docker build: %s", msg.Error)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read build output: %w", err)
	}

	// экспортируем образ и стримим в MinIO, ретрай на случай инфраструктурных проблем
	// ImageSave вызывается внутри ретрая, тк стрим нельзя перечитать повторно
	err = retry.WithRetry(retry.DefaultAttempts, retry.DefaultWait, func() error {
		imageReader, err := b.docker.ImageSave(ctx, tags)
		if err != nil {
			return err
		}
		defer func() { _ = imageReader.Close() }()

		_, err = b.s3.PutObject(ctx, BucketArtifacts, job.ID.String()+".tar", imageReader, UnknownSize,
			minio.PutObjectOptions{ContentType: ContentTypeXTar},
		)
		return err
	})
	if err != nil {
		return fmt.Errorf("upload artifact: %w", err)
	}

	// удаляем локальный образ из docker daemon чтобы не забивать диск
	// некритично - логируем и продолжаем если не получилось
	if _, err = b.docker.ImageRemove(ctx, tags[0], image.RemoveOptions{Force: true}); err != nil {
		slog.Warn("failed to remove docker image",
			slog.String("job_id", job.ID.String()),
			slog.Any("error", err),
		)
	}

	return nil
}
