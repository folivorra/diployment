package coordinator

import (
	"context"
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

type coordinatorService struct {
	repo JobRepository
	jd   JobDispatсher
}

func NewCoordService(repo JobRepository, jd JobDispatсher) *coordinatorService {
	return &coordinatorService{repo: repo, jd: jd}
}

// HandleBuildTriggered обрабатывает событие builds.triggered и отправляет событие jobs.dispatch.
func (c *coordinatorService) HandleBuildTriggered(ctx context.Context, event model.BuildEvent) error {
	job := model.Job{
		ProjectID: event.ProjectID,
		Branch:    event.Branch,
		CloneURL:  event.CloneURL,
		CommitSHA: event.CommitSHA,
		CommitMsg: event.CommitMsg,
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
	return c.repo.UpdateState(ctx, event.JobID, model.StatusRunning, nil)
}

// HandleJobFinished обрабатывает событие jobs.finished.
func (c *coordinatorService) HandleJobFinished(ctx context.Context, event model.JobFinishedEvent) error {
	return c.repo.UpdateState(ctx, event.JobID, event.Status, new(time.Now()))
}
