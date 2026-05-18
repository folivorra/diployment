package service

import (
	"context"
	"fmt"

	"github.com/folivorra/diployment/internal/model"
	"github.com/google/uuid"
)

type JobViewer interface {
	ListByProjectID(ctx context.Context, projectID uuid.UUID) ([]*model.Job, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Job, error)
}

type ProjectGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
}

type jobService struct {
	jobs     JobViewer
	projects ProjectGetter
}

func NewJobService(jobs JobViewer, projects ProjectGetter) *jobService {
	return &jobService{jobs: jobs, projects: projects}
}

func (s *jobService) ListByProject(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) ([]*model.Job, error) {
	project, err := s.projects.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	if project.UserID != userID {
		return nil, ErrForbidden
	}

	return s.jobs.ListByProjectID(ctx, projectID)
}

func (s *jobService) GetByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*model.Job, error) {
	job, err := s.jobs.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	project, err := s.projects.GetByID(ctx, job.ProjectID)
	if err != nil {
		return nil, err
	}

	if project.UserID != userID {
		return nil, ErrForbidden
	}

	return job, nil
}
