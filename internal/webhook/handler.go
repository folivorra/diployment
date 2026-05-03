package webhook

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/pkg/crypto/aesgcm"
	"github.com/folivorra/diployment/pkg/crypto/hmac"

	"github.com/labstack/echo/v4"
)

const (
	shaHexLen          = 40
	shaSignaturePrefix = "sha256="
	shaSignatureHeader = "X-Hub-Signature-256"

	branchPrefix = "refs/heads/"
)

type webhookRequest struct {
	After string `json:"after"`
	Ref   string `json:"ref"`

	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`

	HeadCommit struct {
		Message string `json:"message"`
	} `json:"head_commit"`
}

func (req webhookRequest) IsValid() bool {
	return req.Ref != "" &&
		req.After != "" &&
		len(req.After) == shaHexLen &&
		req.Repository.FullName != "" &&
		strings.Contains(req.Repository.FullName, "/")
}

type ProjectGetter interface {
	GetByFullName(ctx context.Context, fullName string) (*model.Project, error)
}

type BuildEventPusher interface {
	PublishBuildEvent(ctx context.Context, event model.BuildEvent) error
}

type handler struct {
	pg  ProjectGetter
	bp  BuildEventPusher
	key []byte
}

func NewWebhookHandler(pg ProjectGetter, bp BuildEventPusher, key []byte) *handler {
	return &handler{pg: pg, bp: bp, key: key}
}

// Webhook принимает запросы со стороны provider, проверяет подпись и собирает+отправляет событие в NATS.
func (h *handler) Webhook(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// обработка первого запроса на проверку вебхука со стороны провайдера
	if c.Request().Header.Get("X-GitHub-Event") == "ping" {
		return c.NoContent(http.StatusOK)
	}

	var req webhookRequest
	if err = json.Unmarshal(body, &req); err != nil || !req.IsValid() {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	project, err := h.pg.GetByFullName(c.Request().Context(), req.Repository.FullName)
	if err != nil {
		slog.Error("get project by repo full name",
			slog.String("repo", req.Repository.FullName),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	decryptedSecret, err := aesgcm.Decrypt(project.WebhookSecret, h.key, []byte("USER-DATA")) // fixme const userdata
	if err != nil {
		slog.Error("decrypt webhook secret",
			slog.String("project_id", project.ID.String()),
			slog.String("repo", req.Repository.FullName),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	signature := c.Request().Header.Get(shaSignatureHeader)
	if signature == "" {
		return echo.NewHTTPError(http.StatusUnauthorized)
	}
	preparedSignature, err := prepareSignature(signature)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	if ok := hmac.Verify(body, decryptedSecret, preparedSignature); !ok {
		slog.Warn("hmac verification failed",
			slog.String("project_id", project.ID.String()),
			slog.String("repo", req.Repository.FullName),
		)
		return echo.NewHTTPError(http.StatusUnauthorized)
	}

	if req.Ref != branchPrefix+project.Branch {
		slog.Debug("branch mismatch, skipping",
			slog.String("ref", req.Ref),
			slog.String("project_id", project.ID.String()),
			slog.String("repo", req.Repository.FullName),
		)
		return c.NoContent(http.StatusOK)
	}

	event := model.BuildEvent{
		ProjectID:    project.ID,
		RepoFullName: project.RepoFullName,
		CloneURL:     project.CloneURL,
		Branch:       project.Branch,
		CommitSHA:    req.After,
		CommitMsg:    req.HeadCommit.Message,
	}
	if err = h.bp.PublishBuildEvent(c.Request().Context(), event); err != nil {
		slog.Error("publish build event",
			slog.String("project_id", project.ID.String()),
			slog.String("repo", req.Repository.FullName),
			slog.String("error", err.Error()),
		)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

func prepareSignature(signature string) ([]byte, error) {
	signature = strings.TrimPrefix(signature, shaSignaturePrefix)
	decodedSignature, err := hex.DecodeString(signature)
	if err != nil {
		return nil, fmt.Errorf("decode signature from hex")
	}
	return decodedSignature, nil
}
