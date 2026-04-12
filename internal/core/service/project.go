package service

import (
	"context"

	"github.com/folivorra/diployment/internal/model"

	"github.com/google/uuid"
)

type ProjectRepository interface {
	Create(ctx context.Context, project *model.Project) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Project, error)
}

type ProjectService struct {
	repo ProjectRepository
}

func NewProjectService(repo ProjectRepository) *ProjectService {
	return &ProjectService{repo: repo}
}
