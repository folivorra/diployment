package service

import (
	"context"

	"github.com/folivorra/diployment/internal/model"

	"github.com/google/uuid"
)

type UserRepository interface {
	Upsert(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type AuthService struct {
	repo UserRepository
}

func NewAuthService(repo UserRepository) *AuthService {
	return &AuthService{repo: repo}
}
