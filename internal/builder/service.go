package builder

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/pkg/retry"
)

// Builder выполняет сборку образа и возвращает ключ лога в S3.
type Builder interface {
	Build(ctx context.Context, event model.BuildDispatchEvent) (string, error)
}

// BuildStatusPublisher публикует статус сборки в NATS.
type BuildStatusPublisher interface {
	PublishBuildsStarted(ctx context.Context, event model.BuildStartedEvent) error
	PublishBuildsFinished(ctx context.Context, event model.BuildFinishedEvent) error
}

type builderService struct {
	id  string
	b   Builder
	pub BuildStatusPublisher
}

// NewBuilderService создаёт сервис с уникальным id воркера.
func NewBuilderService(id string, b Builder, pub BuildStatusPublisher) *builderService {
	return &builderService{id: id, b: b, pub: pub}
}

// Execute оркестрирует жизненный цикл сборки: уведомляет о старте, запускает сборку и публикует итог.
func (s *builderService) Execute(ctx context.Context, event model.BuildDispatchEvent) error {
	if err := retry.WithRetry(ctx, retry.DefaultAttempts, retry.DefaultWait, func() error {
		return s.pub.PublishBuildsStarted(ctx, model.BuildStartedEvent{
			JobID:     event.JobID,
			ProjectID: event.ProjectID,
			WorkerID:  s.id,
		})
	}); err != nil {
		return fmt.Errorf("publish builds started: %w", err)
	}

	status := model.StatusSuccess
	logURL, buildErr := s.b.Build(ctx, event)
	errMsg := ""
	if buildErr != nil {
		slog.Error("build job failed", slog.Any("error", buildErr))
		status = model.StatusFailed
		errMsg = buildErr.Error()
	}

	if err := retry.WithRetry(ctx, retry.DefaultAttempts, retry.DefaultWait, func() error {
		return s.pub.PublishBuildsFinished(ctx, model.BuildFinishedEvent{
			JobID:     event.JobID,
			ProjectID: event.ProjectID,
			LogURL:    logURL,
			Status:    status,
			Error:     errMsg,
		})
	}); err != nil {
		return fmt.Errorf("publish builds finished: %w", err)
	}

	return nil
}
