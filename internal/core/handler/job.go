package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	GetLog(ctx context.Context, userID uuid.UUID, jobID uuid.UUID, phase model.Phase) (io.ReadCloser, error)
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
	ID               uuid.UUID    `json:"id"`
	Status           model.Status `json:"status"`
	CommitSHA        string       `json:"commit_sha"`
	CommitMsg        string       `json:"commit_msg"`
	CreatedAt        time.Time    `json:"created_at"`
	BuildLogURL      string       `json:"build_log_url,omitempty"`
	BuildStartedAt   *time.Time   `json:"build_started_at,omitempty"`
	BuildFinishedAt  *time.Time   `json:"build_finished_at,omitempty"`
	DeployLogURL     string       `json:"deploy_log_url,omitempty"`
	DeployStartedAt  *time.Time   `json:"deploy_started_at,omitempty"`
	DeployFinishedAt *time.Time   `json:"deploy_finished_at,omitempty"`
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
			ID:               j.ID,
			Status:           j.Status,
			CommitSHA:        j.CommitSHA,
			CommitMsg:        j.CommitMsg,
			CreatedAt:        j.CreatedAt,
			BuildLogURL:      j.BuildLogURL,
			BuildStartedAt:   j.BuildStartedAt,
			BuildFinishedAt:  j.BuildFinishedAt,
			DeployLogURL:     j.DeployLogURL,
			DeployStartedAt:  j.DeployStartedAt,
			DeployFinishedAt: j.DeployFinishedAt,
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

	c.Response().Header().Set("Content-Type", "text/event-stream") // стримим события (логи и обновление статуса)
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	writeStatus := func(status model.Status, phase model.Phase, errStr string) {
		data, _ := json.Marshal(struct {
			Status model.Status `json:"status"`
			Phase  model.Phase  `json:"phase"`
			Error  string       `json:"error,omitempty"`
		}{Status: status, Phase: phase, Error: errStr})
		_, _ = fmt.Fprintf(c.Response(), StatusFormat, data)
		c.Response().Flush()
	}
	writeLog := func(line string, phase model.Phase) {
		data, _ := json.Marshal(struct {
			Line  string      `json:"line"`
			Phase model.Phase `json:"phase"`
		}{Line: line, Phase: phase})
		_, _ = fmt.Fprintf(c.Response(), LogFormat, data)
		c.Response().Flush()
	}

	// определяем текущую фазу по полям джобы из БД
	// нужно чтобы отправить правильный начальный статус клиенту
	currentPhase := func() model.Phase {
		if job.DeployStartedAt != nil {
			return model.PhaseDeploy
		}
		if job.BuildStartedAt != nil {
			return model.PhaseBuild
		}
		return ""
	}()

	writeStatus(job.Status, currentPhase, "")

	// если джоба уже завершена - подписываемся только на логи
	// NATS отдаст все старые строки из буфера (DeliverAllPolicy)
	// и мы закроем стрим когда придёт Done нужной фазы
	alreadyDone := job.Status == model.StatusSuccess || job.Status == model.StatusFailed

	logCh, stopLogs, err := h.lgs.SubscribeLogs(c.Request().Context(), jobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	defer stopLogs()

	// nil-канал в select никогда не выбирается
	var notifCh <-chan model.JobNotifyEvent
	stopNotif := func() {}
	if !alreadyDone {
		notifCh, stopNotif, err = h.ntf.SubscribeNotifications(c.Request().Context(), jobID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}
	}
	defer stopNotif()

	// gotTerminalStatus - знаем что джоба завершена (либо из БД, либо пришло по notifCh)
	// buildDone/deployDone - пришёл Done-сентинель соответствующей фазы
	gotTerminalStatus := alreadyDone
	buildDone := false
	deployDone := false

	for {
		select {
		case <-c.Request().Context().Done():
			return nil

		case event, ok := <-notifCh:
			if !ok {
				return nil
			}
			writeStatus(event.Status, event.Phase, event.Error)
			if event.Status == model.StatusSuccess || event.Status == model.StatusFailed {
				gotTerminalStatus = true
			}
			if gotTerminalStatus && (buildDone || deployDone) {
				return nil
			}

		case line, ok := <-logCh:
			if !ok {
				return nil
			}
			if !line.Done {
				writeLog(line.Line, line.Phase)
				continue
			}
			// Done-сентинель - ждём финального статуса в обоих случаях,
			// так как notify может прийти после Done из-за порядка публикации в NATS
			if line.Phase == model.PhaseDeploy {
				deployDone = true
			} else {
				buildDone = true
			}
			if gotTerminalStatus {
				return nil
			}
		}
	}
}

// GetLog отдаёт персистентный лог джобы из MinIO в виде text/plain стрима.
func (h *jobHandler) GetLog(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user id not found or empty")
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	phase := model.Phase(c.Param("phase"))
	if !phase.IsValid() {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid phase, must be 'build' or 'deploy'")
	}

	rc, err := h.svc.GetLog(c.Request().Context(), userID, jobID, phase)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			return echo.NewHTTPError(http.StatusForbidden, "access denied")
		case errors.Is(err, service.ErrLogUnavailable):
			return echo.NewHTTPError(http.StatusNotFound, "log not available")
		case errors.Is(err, postgres.ErrJobNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "job not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}
	}
	defer rc.Close()

	c.Response().Header().Set(echo.HeaderContentType, "text/plain; charset=utf-8")
	c.Response().WriteHeader(http.StatusOK)
	_, _ = io.Copy(c.Response(), rc)

	return nil
}
