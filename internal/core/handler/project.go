package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/folivorra/diployment/internal/core/service"
	"github.com/folivorra/diployment/internal/pgpool"
	"github.com/labstack/echo/v4"

	"github.com/google/uuid"
)

type ProjectService interface {
	Import(ctx context.Context, userID *uuid.UUID, input service.ImportProjectInput) error
}

type projectHandler struct {
	srv ProjectService
}

func NewProjectHandler(srv ProjectService) *projectHandler {
	return &projectHandler{srv: srv}
}

func (p *projectHandler) Import(c echo.Context) error {
	rawOwnerID := c.Get("user_id")

	ownerID, ok := rawOwnerID.(*uuid.UUID)
	if !ok || ownerID == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user id not found or empty")
	}

	var input service.ImportProjectInput
	err := c.Bind(&input)
	if err != nil || !input.IsValid() {
		return echo.NewHTTPError(http.StatusBadRequest, "project input invalid")
	}

	err = p.srv.Import(c.Request().Context(), ownerID, input)
	if err != nil {
		slog.Error("import project failed", slog.String("error", err.Error()))

		switch {
		case errors.Is(err, pgpool.ErrProjectAlreadyExist):
			return echo.NewHTTPError(http.StatusConflict, "project already imported")
		case errors.Is(err, service.ErrProviderAPI):
			return echo.NewHTTPError(http.StatusBadGateway, "provider API is unavailable")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}
	}

	return c.NoContent(http.StatusCreated)
}
