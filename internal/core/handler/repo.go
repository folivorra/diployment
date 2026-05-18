package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/folivorra/diployment/internal/core/service"
	"github.com/folivorra/diployment/internal/model"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type RepoService interface {
	ListUserReposByID(ctx context.Context, userID uuid.UUID) ([]*model.Repository, error)
	ListRepoBranchesByID(ctx context.Context, userID uuid.UUID, repoFullName string) ([]string, error)
}

type repoHandler struct {
	srv RepoService
}

func NewRepoHandler(srv RepoService) *repoHandler {
	return &repoHandler{srv: srv}
}

// ListRepos возвращает список репозиториев из провайдер-хаба пользователя.
func (r *repoHandler) ListRepos(c echo.Context) error {
	rawUserID := c.Get("user_id")

	userID, ok := rawUserID.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user id not found or empty")
	}

	repos, err := r.srv.ListUserReposByID(c.Request().Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProviderAPI):
			return echo.NewHTTPError(http.StatusBadGateway, "provider API is unavailable")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}
	}

	return c.JSON(http.StatusOK, repos)
}

// ListBranches возвращает список веток репозитория.
func (r *repoHandler) ListBranches(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user id not found or empty")
	}

	repoFullName := c.QueryParam("full_name")
	if repoFullName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "full_name query param is required")
	}

	branches, err := r.srv.ListRepoBranchesByID(c.Request().Context(), userID, repoFullName)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProviderAPI):
			return echo.NewHTTPError(http.StatusBadGateway, "provider API is unavailable")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}
	}

	return c.JSON(http.StatusOK, branches)
}
