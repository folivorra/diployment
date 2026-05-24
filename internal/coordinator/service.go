package coordinator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/pkg/retry"

	"github.com/google/uuid"
)

// JobRepository хранит и обновляет состояние джоб.
type JobRepository interface {
	Create(ctx context.Context, job *model.Job) (uuid.UUID, error)
	Delete(ctx context.Context, id uuid.UUID) error

	UpdateBuildStarted(ctx context.Context, id uuid.UUID, startedAt time.Time) error
	UpdateBuildFinished(ctx context.Context, id uuid.UUID, status model.Status, buildLogURL *string, finishedAt time.Time) error
	UpdateDeployStarted(ctx context.Context, id uuid.UUID, startedAt time.Time) error
	UpdateDeployFinished(ctx context.Context, id uuid.UUID, status model.Status, deployLogURL *string, finishedAt time.Time) error

	FailStale(ctx context.Context, olderThan time.Duration) ([]model.StaleJob, error)
}

// ProjectGetter читает данные проекта, нужен для диспатча деплоя.
type ProjectGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
}

// BuildDispatcher отправляет задачу на сборку в NATS.
type BuildDispatcher interface {
	PublishBuildsDispatch(ctx context.Context, event model.BuildDispatchEvent) error
}

// DeployDispatcher отправляет задачу на деплой в NATS.
type DeployDispatcher interface {
	PublishDeployDispatch(ctx context.Context, event model.DeployDispatchEvent) error
}

// JobNotifier публикует уведомление о статусе джобы для SSE-клиентов.
type JobNotifier interface {
	PublishJobsNotify(ctx context.Context, event model.JobNotifyEvent) error
}

type coordinatorService struct {
	repo JobRepository
	proj ProjectGetter
	bd   BuildDispatcher
	dd   DeployDispatcher
	jn   JobNotifier
}

// NewCoordService создаёт координатор с зависимостями.
func NewCoordService(repo JobRepository, proj ProjectGetter, bd BuildDispatcher, dd DeployDispatcher, jn JobNotifier) *coordinatorService {
	return &coordinatorService{repo: repo, proj: proj, bd: bd, dd: dd, jn: jn}
}

// RunStaleJobsWatchdog периодически ищет зависшие джобы и переводит их в failed.
// Для каждой такой джобы публикует JobNotifyEvent, чтобы разблокировать SSE-клиентов.
func (c *coordinatorService) RunStaleJobsWatchdog(ctx context.Context, interval, olderThan time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stale, err := c.repo.FailStale(ctx, olderThan)
			if err != nil {
				slog.Error("stale jobs watchdog", slog.Any("error", err))
				continue
			}
			for _, job := range stale {
				slog.Warn("stale job failed by watchdog", slog.String("job_id", job.ID.String()))
				if err := c.jn.PublishJobsNotify(ctx, model.JobNotifyEvent{
					JobID:  job.ID,
					Status: model.StatusFailed,
					Phase:  job.Phase,
					Error:  "job timed out: builder did not respond",
				}); err != nil {
					slog.Error("publish stale job notify",
						slog.String("job_id", job.ID.String()),
						slog.Any("error", err),
					)
				}
			}
		}
	}
}

// HandleJobTriggered обрабатывает событие jobs.triggered и отправляет событие builds.dispatch.
func (c *coordinatorService) HandleJobTriggered(ctx context.Context, event model.JobTriggeredEvent) error {
	job := model.Job{
		ProjectID:      event.ProjectID,
		Branch:         event.Branch,
		CloneURL:       event.CloneURL,
		EncryptedToken: event.EncryptedToken,
		CommitSHA:      event.CommitSHA,
		CommitMsg:      event.CommitMsg,
	}

	id, err := c.repo.Create(ctx, &job)
	if err != nil {
		return err
	}
	job.ID = id

	if err := retry.WithRetry(ctx, retry.DefaultAttempts, retry.DefaultWait, func() error {
		return c.bd.PublishBuildsDispatch(ctx, model.BuildDispatchEvent{
			JobID:          id,
			ProjectID:      event.ProjectID,
			CloneURL:       event.CloneURL,
			Branch:         event.Branch,
			EncryptedToken: event.EncryptedToken,
		})
	}); err != nil {
		// компенсация: удаляем джобу из DB, чтобы она не зависла в pending без воркера
		if delErr := c.repo.Delete(ctx, id); delErr != nil {
			slog.Error("compensating delete failed after dispatch error",
				slog.String("job_id", id.String()),
				slog.Any("error", delErr),
			)
		}
		return err
	}
	return nil
}

// HandleBuildStarted обрабатывает событие builds.started.
func (c *coordinatorService) HandleBuildStarted(ctx context.Context, event model.BuildStartedEvent) error {
	if err := c.repo.UpdateBuildStarted(ctx, event.JobID, time.Now()); err != nil {
		return err
	}

	if err := retry.WithRetry(ctx, retry.DefaultAttempts, retry.DefaultWait, func() error {
		return c.jn.PublishJobsNotify(ctx, model.JobNotifyEvent{
			JobID:  event.JobID,
			Status: model.StatusBuilding,
			Phase:  model.PhaseBuild,
		})
	}); err != nil {
		slog.Error("publish job notify",
			slog.String("job_id", event.JobID.String()),
			slog.Any("error", err),
		)
	}

	return nil
}

// HandleBuildFinished обрабатывает событие builds.finished.
// При успехе диспатчит деплой, при неудаче терминирует джобу.
func (c *coordinatorService) HandleBuildFinished(ctx context.Context, event model.BuildFinishedEvent) error {
	var logURL *string
	if event.LogURL != "" {
		logURL = &event.LogURL
	}

	if event.Status == model.StatusFailed {
		if err := c.repo.UpdateBuildFinished(ctx, event.JobID, model.StatusFailed, logURL, time.Now()); err != nil {
			return err
		}
		if err := retry.WithRetry(ctx, retry.DefaultAttempts, retry.DefaultWait, func() error {
			return c.jn.PublishJobsNotify(ctx, model.JobNotifyEvent{
				JobID:  event.JobID,
				Status: model.StatusFailed,
				Phase:  model.PhaseBuild,
				Error:  event.Error,
			})
		}); err != nil {
			slog.Error("publish job notify",
				slog.String("job_id", event.JobID.String()),
				slog.Any("error", err),
			)
		}
		return nil
	}

	proj, err := c.proj.GetByID(ctx, event.ProjectID)
	if err != nil {
		return fmt.Errorf("get project for deploy dispatch: %w", err)
	}

	dispatchErr := retry.WithRetry(ctx, retry.DefaultAttempts, retry.DefaultWait, func() error {
		return c.dd.PublishDeployDispatch(ctx, model.DeployDispatchEvent{
			JobID:            event.JobID,
			ProjectID:        event.ProjectID,
			ImageArtifactKey: event.JobID.String() + ".tar",
			SSHHost:          proj.SSHHost,
			SSHPort:          proj.SSHPort,
			SSHUser:          proj.SSHUser,
			EncryptedSSHKey:  proj.SSHKey,
			RestartCmd:       proj.DeployRestartCmd,
			Workdir:          proj.DeployWorkdir,
		})
	})

	finalBuildStatus := model.StatusDeploying
	if dispatchErr != nil {
		slog.Error("dispatch deploy failed, terminating job",
			slog.String("job_id", event.JobID.String()),
			slog.Any("error", dispatchErr),
		)
		finalBuildStatus = model.StatusFailed
	}

	if err := c.repo.UpdateBuildFinished(ctx, event.JobID, finalBuildStatus, logURL, time.Now()); err != nil {
		return err
	}

	if dispatchErr != nil {
		if err := retry.WithRetry(ctx, retry.DefaultAttempts, retry.DefaultWait, func() error {
			return c.jn.PublishJobsNotify(ctx, model.JobNotifyEvent{
				JobID:  event.JobID,
				Status: model.StatusFailed,
				Phase:  model.PhaseBuild,
				Error:  "failed to dispatch deploy",
			})
		}); err != nil {
			slog.Error("publish job notify",
				slog.String("job_id", event.JobID.String()),
				slog.Any("error", err),
			)
		}
		return nil
	}

	if err := retry.WithRetry(ctx, retry.DefaultAttempts, retry.DefaultWait, func() error {
		return c.jn.PublishJobsNotify(ctx, model.JobNotifyEvent{
			JobID:  event.JobID,
			Status: model.StatusDeploying,
			Phase:  model.PhaseDeploy,
		})
	}); err != nil {
		slog.Error("publish job notify",
			slog.String("job_id", event.JobID.String()),
			slog.Any("error", err),
		)
	}

	return nil
}

// HandleDeployStarted обрабатывает событие deploys.started.
func (c *coordinatorService) HandleDeployStarted(ctx context.Context, event model.DeployStartedEvent) error {
	if err := c.repo.UpdateDeployStarted(ctx, event.JobID, time.Now()); err != nil {
		return err
	}

	if err := retry.WithRetry(ctx, retry.DefaultAttempts, retry.DefaultWait, func() error {
		return c.jn.PublishJobsNotify(ctx, model.JobNotifyEvent{
			JobID:  event.JobID,
			Status: model.StatusDeploying,
			Phase:  model.PhaseDeploy,
		})
	}); err != nil {
		slog.Error("publish job notify",
			slog.String("job_id", event.JobID.String()),
			slog.Any("error", err),
		)
	}

	return nil
}

// HandleDeployFinished обрабатывает событие deploys.finished.
func (c *coordinatorService) HandleDeployFinished(ctx context.Context, event model.DeployFinishedEvent) error {
	var logURL *string
	if event.LogURL != "" {
		logURL = &event.LogURL
	}

	if err := c.repo.UpdateDeployFinished(ctx, event.JobID, event.Status, logURL, time.Now()); err != nil {
		return err
	}

	if err := retry.WithRetry(ctx, retry.DefaultAttempts, retry.DefaultWait, func() error {
		return c.jn.PublishJobsNotify(ctx, model.JobNotifyEvent{
			JobID:  event.JobID,
			Status: event.Status,
			Phase:  model.PhaseDeploy,
			Error:  event.Error,
		})
	}); err != nil {
		slog.Error("publish job notify",
			slog.String("job_id", event.JobID.String()),
			slog.Any("error", err),
		)
	}

	return nil
}
