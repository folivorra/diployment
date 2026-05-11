package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/pkg/retry"
)

type Builder interface {
	Build(ctx context.Context, job model.Job) error
}

type JobStatusPublisher interface {
	PublishJobStarted(ctx context.Context, event model.JobStartedEvent) error
	PublishJobFinished(ctx context.Context, event model.JobFinishedEvent) error
}

type workerService struct {
	id  string
	b   Builder
	pub JobStatusPublisher
}

func NewWorkerService(id string, b Builder, pub JobStatusPublisher) *workerService {
	return &workerService{id: id, b: b, pub: pub}
}

// ExecuteJob оркестрирует жизненный цикл джобы: уведомляет о старте, запускает сборку и публикует итог.
// Возвращает ошибку только если не удалось уведомить о старте - в этом случае консьюмер делает Nak.
func (s *workerService) ExecuteJob(ctx context.Context, job model.Job) error {
	if err := retry.WithRetry(retry.DefaultAttempts, retry.DefaultWait, func() error {
		return s.pub.PublishJobStarted(ctx, model.JobStartedEvent{
			JobID:     job.ID,
			ProjectID: job.ProjectID,
			WorkerID:  s.id,
		})
	}); err != nil {
		return fmt.Errorf("publish job started: %w", err)
	}

	status := model.StatusSuccess
	buildErr := s.b.Build(ctx, job)
	errMsg := ""
	if buildErr != nil {
		slog.Error("build job failed", slog.Any("error", buildErr))
		status = model.StatusFailed
		errMsg = buildErr.Error()
	}

	if err := retry.WithRetry(retry.DefaultAttempts, retry.DefaultWait, func() error {
		return s.pub.PublishJobFinished(ctx, model.JobFinishedEvent{
			JobID:     job.ID,
			ProjectID: job.ProjectID,
			Status:    status,
			Error:     errMsg,
		})
	}); err != nil {
		slog.Error("publish job finish failed", slog.Any("error", err))
	}

	return nil
}
