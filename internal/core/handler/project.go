package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/folivorra/diployment/internal/core/service"
	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/internal/repository/postgres"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ProjectService interface {
	Import(ctx context.Context, userID uuid.UUID, input service.ImportProjectInput) error
	List(ctx context.Context, userID uuid.UUID) ([]*model.Project, error)
}

type projectHandler struct {
	srv ProjectService
}

func NewProjectHandler(srv ProjectService) *projectHandler {
	return &projectHandler{srv: srv}
}

// Import импортирует репозиторий GitHub как проект и регистрирует вебхук на стороне провайдера.
func (p *projectHandler) Import(c echo.Context) error {
	ownerID, ok := c.Get("user_id").(uuid.UUID)
	if !ok || ownerID == uuid.Nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user id not found or empty")
	}

	var input service.ImportProjectInput
	err := c.Bind(&input)
	if err != nil || !input.IsValid() {
		return echo.NewHTTPError(http.StatusBadRequest, "project input invalid")
	}

	err = p.srv.Import(c.Request().Context(), ownerID, input)
	if err != nil {
		slog.Error("import project failed", slog.Any("error", err))

		switch {
		case errors.Is(err, postgres.ErrProjectAlreadyExist):
			return echo.NewHTTPError(http.StatusConflict, "project already imported")
		case errors.Is(err, service.ErrProviderAPI):
			return echo.NewHTTPError(http.StatusBadGateway, "provider API is unavailable")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}
	}

	return c.NoContent(http.StatusCreated)
}

// List возвращает список проектов текущего пользователя.
func (p *projectHandler) List(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user id not found or empty")
	}

	projects, err := p.srv.List(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, projects)
}
