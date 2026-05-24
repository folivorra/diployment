package deployer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/pkg/retry"
)

// Deployer выполняет деплой на удалённый сервер и возвращает ключ лога в S3.
type Deployer interface {
	Deploy(ctx context.Context, event model.DeployDispatchEvent) (string, error)
}

// DeployStatusPublisher публикует статус деплоя в NATS.
type DeployStatusPublisher interface {
	PublishDeployStarted(ctx context.Context, event model.DeployStartedEvent) error
	PublishDeployFinished(ctx context.Context, event model.DeployFinishedEvent) error
}

type deployerService struct {
	id  string
	d   Deployer
	pub DeployStatusPublisher
}

// NewDeployerService создаёт сервис с уникальным id воркера.
func NewDeployerService(id string, d Deployer, pub DeployStatusPublisher) *deployerService {
	return &deployerService{id: id, d: d, pub: pub}
}

// Execute оркестрирует жизненный цикл деплоя: уведомляет о старте, запускает деплой и публикует итог.
func (s *deployerService) Execute(ctx context.Context, event model.DeployDispatchEvent) error {
	if err := retry.WithRetry(ctx, retry.DefaultAttempts, retry.DefaultWait, func() error {
		return s.pub.PublishDeployStarted(ctx, model.DeployStartedEvent{
			JobID:     event.JobID,
			ProjectID: event.ProjectID,
			WorkerID:  s.id,
		})
	}); err != nil {
		return fmt.Errorf("publish deploy started: %w", err)
	}

	status := model.StatusSuccess
	logURL, deployErr := s.d.Deploy(ctx, event)
	errMsg := ""
	if deployErr != nil {
		slog.Error("deploy job failed", slog.Any("error", deployErr))
		status = model.StatusFailed
		errMsg = deployErr.Error()
	}

	if err := retry.WithRetry(ctx, retry.DefaultAttempts, retry.DefaultWait, func() error {
		return s.pub.PublishDeployFinished(ctx, model.DeployFinishedEvent{
			JobID:     event.JobID,
			ProjectID: event.ProjectID,
			LogURL:    logURL,
			Status:    status,
			Error:     errMsg,
		})
	}); err != nil {
		return fmt.Errorf("publish deploy finished: %w", err)
	}

	return nil
}
