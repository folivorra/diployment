package handler

import (
	"context"
	"errors"
	"fmt"
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
	GetByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*model.Job, error)
}

type JobEventSubscriber interface {
	Subscribe(ctx context.Context, jobID uuid.UUID) (<-chan model.JobNotifyEvent, func(), error)
}

type jobHandler struct {
	svc JobService
	sub JobEventSubscriber
}

func NewJobHandler(svc JobService, sub JobEventSubscriber) *jobHandler {
	return &jobHandler{svc: svc, sub: sub}
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

// Events стримит статусы джобы через SSE.
func (h *jobHandler) Events(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user id not found or empty")
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	job, err := h.svc.GetByID(c.Request().Context(), userID, jobID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			return echo.NewHTTPError(http.StatusForbidden, "access denied")
		case errors.Is(err, postgres.ErrJobNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "job not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}
	}

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	writeEvent := func(status model.Status) {
		_, _ = fmt.Fprintf(c.Response(), "data: {\"status\":\"%s\"}\n\n", status)
		c.Response().Flush()
	}

	writeEvent(job.Status)
	if job.Status == model.StatusSuccess || job.Status == model.StatusFailed {
		return nil
	}

	ch, stop, err := h.sub.Subscribe(c.Request().Context(), jobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	defer stop()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case event, ok := <-ch:
			if !ok {
				return nil
			}
			writeEvent(event.Status)
			if event.Status == model.StatusSuccess || event.Status == model.StatusFailed {
				return nil
			}
		}
	}
}
