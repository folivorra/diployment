package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/folivorra/diployment/internal/core/service"
	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/internal/repository/postgres"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type JobService interface {
	ListByProject(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) ([]*model.Job, error)
}

type jobHandler struct {
	svc JobService
}

func NewJobHandler(svc JobService) *jobHandler {
	return &jobHandler{svc: svc}
}

type jobResponse struct {
	ID         uuid.UUID    `json:"id"`
	Status     model.Status `json:"status"`
	CommitSHA  string       `json:"commit_sha"`
	CommitMsg  string       `json:"commit_msg"`
	LogURL     string       `json:"log_url,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
	FinishedAt *time.Time   `json:"finished_at,omitempty"`
}

// ListByProject возвращает список джоб для указанного проекта.
func (h *jobHandler) ListByProject(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user id not found or empty")
	}

	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}

	jobs, err := h.svc.ListByProject(c.Request().Context(), userID, projectID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			return echo.NewHTTPError(http.StatusForbidden, "access denied")
		case errors.Is(err, postgres.ErrProjectNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "project not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}
	}

	resp := make([]jobResponse, len(jobs))
	for i, j := range jobs {
		resp[i] = jobResponse{
			ID:         j.ID,
			Status:     j.Status,
			CommitSHA:  j.CommitSHA,
			CommitMsg:  j.CommitMsg,
			LogURL:     j.LogURL,
			CreatedAt:  j.CreatedAt,
			FinishedAt: j.FinishedAt,
		}
	}

	return c.JSON(http.StatusOK, resp)
}
