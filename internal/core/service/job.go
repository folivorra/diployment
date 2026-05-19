package service

import (
	"context"

	"github.com/folivorra/diployment/internal/model"

	"github.com/google/uuid"
)

type JobViewer interface {
	GetByIDWithOwner(ctx context.Context, id uuid.UUID) (*model.Job, uuid.UUID, error)
	ListByProjectIDWithOwner(ctx context.Context, projectID uuid.UUID) ([]*model.Job, uuid.UUID, error)
}

type jobService struct {
	jobs JobViewer
}

func NewJobService(jobs JobViewer) *jobService {
	return &jobService{jobs: jobs}
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
