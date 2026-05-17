package service

import (
	"context"
	"fmt"

	"github.com/folivorra/diployment/internal/model"
	"github.com/google/uuid"
)

type JobsLister interface {
	ListByProjectID(ctx context.Context, projectID uuid.UUID) ([]*model.Job, error)
}

type ProjectGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
}

type jobService struct {
	jobs     JobsLister
	projects ProjectGetter
}

func NewJobService(jobs JobsLister, projects ProjectGetter) *jobService {
	return &jobService{jobs: jobs, projects: projects}
}

// ListByProject возвращает список джоб проекта, предварительно проверяя, что проект принадлежит пользователю.
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
