package coordinator

import (
	"context"
	"log/slog"
	"time"

	"github.com/folivorra/diployment/internal/model"

	"github.com/google/uuid"
)

type JobRepository interface {
	Create(ctx context.Context, job *model.Job) (uuid.UUID, error)
	UpdateState(ctx context.Context, id uuid.UUID, status model.Status, finishedAt *time.Time) error
}

type JobDispatсher interface {
	PublishJobDispatch(ctx context.Context, event model.Job) error
}

type JobNotifier interface {
	PublishJobNotify(ctx context.Context, event model.JobNotifyEvent) error
}

type coordinatorService struct {
	repo JobRepository
	jd   JobDispatсher
	jn   JobNotifier
}

func NewCoordService(repo JobRepository, jd JobDispatсher, jn JobNotifier) *coordinatorService {
	return &coordinatorService{repo: repo, jd: jd, jn: jn}
}

// HandleBuildTriggered обрабатывает событие builds.triggered и отправляет событие jobs.dispatch.
func (c *coordinatorService) HandleBuildTriggered(ctx context.Context, event model.BuildEvent) error {
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

	return c.jd.PublishJobDispatch(ctx, job)
}

// HandleJobStarted обрабатывает событие jobs.started.
func (c *coordinatorService) HandleJobStarted(ctx context.Context, event model.JobStartedEvent) error {
	if err := c.repo.UpdateState(ctx, event.JobID, model.StatusRunning, nil); err != nil {
		return err
	}

	if err := c.jn.PublishJobNotify(ctx, model.JobNotifyEvent{
		JobID:  event.JobID,
		Status: model.StatusRunning,
	}); err != nil {
		slog.Error("publish job notify",
			slog.String("job_id", event.JobID.String()),
			slog.Any("error", err),
		)
	}

	return nil
}

// HandleJobFinished обрабатывает событие jobs.finished.
func (c *coordinatorService) HandleJobFinished(ctx context.Context, event model.JobFinishedEvent) error {
	if err := c.repo.UpdateState(ctx, event.JobID, event.Status, new(time.Now())); err != nil {
		return err
	}

	if err := c.jn.PublishJobNotify(ctx, model.JobNotifyEvent{
		JobID:  event.JobID,
		Status: event.Status,
		Error:  event.Error,
	}); err != nil {
		slog.Error("publish job notify",
			slog.String("job_id", event.JobID.String()),
			slog.Any("error", err),
		)
	}

	return nil
}
