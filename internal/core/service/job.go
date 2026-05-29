package service

import (
	"context"
	"errors"
	"io"

	"github.com/folivorra/diployment/internal/model"
	miniorepo "github.com/folivorra/diployment/internal/repository/minio"

	"github.com/google/uuid"
)

type JobViewer interface {
	GetByIDWithOwner(ctx context.Context, id uuid.UUID) (*model.Job, uuid.UUID, error)
	ListByProjectIDWithOwner(ctx context.Context, projectID uuid.UUID) ([]*model.Job, uuid.UUID, error)
}

type LogRepository interface {
	Get(ctx context.Context, jobID uuid.UUID, phase model.Phase) (io.ReadCloser, error)
}

type jobService struct {
	jobs JobViewer
	logs LogRepository
}

func NewJobService(jobs JobViewer, logs LogRepository) *jobService {
	return &jobService{jobs: jobs, logs: logs}
}

func (s *jobService) ListByProject(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) ([]*model.Job, error) {
	jobs, ownerID, err := s.jobs.ListByProjectIDWithOwner(ctx, projectID)
	if err != nil {
		return nil, err
	}

	if ownerID != userID {
		return nil, ErrForbidden
	}

	return jobs, nil
}

func (s *jobService) GetByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*model.Job, error) {
	job, ownerID, err := s.jobs.GetByIDWithOwner(ctx, id)
	if err != nil {
		return nil, err
	}

	if ownerID != userID {
		return nil, ErrForbidden
	}

	return job, nil
}

func (s *jobService) GetLog(ctx context.Context, userID uuid.UUID, jobID uuid.UUID, phase model.Phase) (io.ReadCloser, error) {
	if !phase.IsValid() {
		return nil, ErrLogUnavailable
	}

	job, ownerID, err := s.jobs.GetByIDWithOwner(ctx, jobID)
	if err != nil {
		return nil, err
	}

	if ownerID != userID {
		return nil, ErrForbidden
	}

	switch phase {
	case model.PhaseBuild:
		if job.BuildLogURL == "" {
			return nil, ErrLogUnavailable
		}
	case model.PhaseDeploy:
		if job.DeployLogURL == "" {
			return nil, ErrLogUnavailable
		}
	}

	rc, err := s.logs.Get(ctx, jobID, phase)
	if err != nil {
		if errors.Is(err, miniorepo.ErrLogNotFound) {
			return nil, ErrLogUnavailable
		}
		return nil, err
	}

	return rc, nil
}
