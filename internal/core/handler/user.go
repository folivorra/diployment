package handler

import (
	"context"
	"net/http"

	"github.com/folivorra/diployment/internal/model"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type UserService interface {
	Me(ctx context.Context, userID uuid.UUID) (*model.User, error)
}

type userHandler struct {
	svc UserService
}

func NewUserHandler(svc UserService) *userHandler {
	return &userHandler{svc: svc}
}

// Me возвращает информацию о текущем авторизованном пользователе.
func (h *userHandler) Me(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user id not found or empty")
	}

	user, err := h.svc.Me(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, user)
}
