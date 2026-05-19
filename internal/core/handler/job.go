package handler

import (
	"context"
	"encoding/json"
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

type JobNotificationsSubscriber interface {
	SubscribeNotifications(ctx context.Context, jobID uuid.UUID) (<-chan model.JobNotifyEvent, func(), error)
}

type JobLogSubscriber interface {
	SubscribeLogs(ctx context.Context, jobID uuid.UUID) (<-chan model.JobLogLine, func(), error)
}

const (
	StatusFormat = "event: status\ndata: %s\n\n"
	LogFormat    = "event: log\ndata: %s\n\n"
)

type jobHandler struct {
	svc JobService
	ntf JobNotificationsSubscriber
	lgs JobLogSubscriber
}

func NewJobHandler(svc JobService, ntf JobNotificationsSubscriber, lgs JobLogSubscriber) *jobHandler {
	return &jobHandler{svc: svc, ntf: ntf, lgs: lgs}
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

// Events стримит статусы и логи джобы через SSE на одном соединении.
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

	writeStatus := func(status model.Status, errStr string) {
		data, _ := json.Marshal(struct {
			Status model.Status `json:"status"`
			Error  string       `json:"error"`
		}{Status: status, Error: errStr})
		_, _ = fmt.Fprintf(c.Response(), StatusFormat, data)
		c.Response().Flush()
	}

	writeLog := func(line string) {
		data, _ := json.Marshal(struct {
			Line string `json:"line"`
		}{Line: line})
		_, _ = fmt.Fprintf(c.Response(), LogFormat, data)
		c.Response().Flush()
	}

	// отправляем начальный статус сразу при подключении
	writeStatus(job.Status, "")

	gotTerminalStatus := job.Status == model.StatusSuccess || job.Status == model.StatusFailed
	gotLogsDone := false

	// подписываемся на логи всегда, DeliverAllPolicy отдаёт буфер прошлых строк + новые
	logCh, stopLogs, err := h.lgs.SubscribeLogs(c.Request().Context(), jobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	defer stopLogs()

	// подписываемся на статусы только если джоба ещё не завершена
	// nil-канал в select никогда не выбирается
	var notifCh <-chan model.JobNotifyEvent
	stopNotif := func() {}
	if !gotTerminalStatus {
		notifCh, stopNotif, err = h.ntf.SubscribeNotifications(c.Request().Context(), jobID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}
	}
	defer stopNotif()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case event, ok := <-notifCh:
			if !ok {
				return nil
			}
			writeStatus(event.Status, event.Error)
			if event.Status == model.StatusSuccess || event.Status == model.StatusFailed {
				gotTerminalStatus = true
			}
			if gotTerminalStatus && gotLogsDone {
				return nil
			}
		case line, ok := <-logCh:
			if !ok {
				return nil
			}
			if line.Done {
				gotLogsDone = true
			} else {
				writeLog(line.Line)
			}
			if gotTerminalStatus && gotLogsDone {
				return nil
			}
		}
	}
}
