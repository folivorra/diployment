package service

import (
	"context"

	"github.com/folivorra/diployment/internal/model"
	"github.com/google/uuid"
)

type UserByIDGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type userService struct {
	repo UserByIDGetter
}

func NewUserService(repo UserByIDGetter) *userService {
	return &userService{repo: repo}
}

// Me возвращает информацию о текущем авторизованном пользователе.
func (s *userService) Me(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	return s.repo.GetByID(ctx, userID)
}
