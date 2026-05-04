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
	ListUserReposByID(ctx context.Context, userID *uuid.UUID) ([]*model.Repository, error)
}

type repoHandler struct {
	srv RepoService
}

func NewRepoHandler(srv RepoService) *repoHandler {
	return &repoHandler{srv: srv}
}

func (r *repoHandler) ListRepos(c echo.Context) error {
	rawUserID := c.Get("user_id")

	userID, ok := rawUserID.(*uuid.UUID)
	if !ok || userID == nil {
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
