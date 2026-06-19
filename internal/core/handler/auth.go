package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"

	"github.com/folivorra/diployment/internal/core/service"

	"github.com/labstack/echo/v4"
)

type AuthService interface {
	GetAuthCodeURL(state string) string
	Authenticate(ctx context.Context, code string) (string, error)
}

type authHandler struct {
	svc         AuthService
	frontendURL string
}

func NewAuthHandler(authSrv AuthService, frontendURL string) *authHandler {
	return &authHandler{
		svc:         authSrv,
		frontendURL: frontendURL,
	}
}

// Login выполняет редирект на провайдера для запроса прав.
func (a *authHandler) Login(c echo.Context) error {
	state := generateState()

	c.SetCookie(&http.Cookie{
		Name:     "state",
		Value:    state,
		MaxAge:   3600,
		Path:     "/",
		Secure:   false, // fixme на проде это ДОЛЖНО быть true
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	url := a.svc.GetAuthCodeURL(state)

	return c.Redirect(http.StatusTemporaryRedirect, url)
}

// Callback обменивает code на JWT токен и устанавливает его в куку.
func (a *authHandler) Callback(c echo.Context) error {
	if err := validateState(c); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "state invalid")
	}

	code := c.QueryParam("code")
	if code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "code is missing")
	}

	jwtToken, err := a.svc.Authenticate(c.Request().Context(), code)
	if err != nil {
		slog.Error("err", slog.String("text", err.Error()))
		switch {
		case errors.Is(err, service.ErrCodeExchange):
			return echo.NewHTTPError(http.StatusForbidden, "authorization code is invalid or expired")
		case errors.Is(err, service.ErrProviderAPI):
			return echo.NewHTTPError(http.StatusBadGateway, "provider API is unavailable")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}
	}

	c.SetCookie(&http.Cookie{
		Name:     "jwt_token",
		Value:    jwtToken,
		HttpOnly: true,
		Secure:   false, // fixme на проде это ДОЛЖНО быть true
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	return c.Redirect(http.StatusTemporaryRedirect, a.frontendURL+"/dashboard")
}

// generateState возвращает рандомно сгенерированную строку в base64 для использования как state.
func generateState() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// validateState валидирует state, сравнивая переданное в cookie и в query.
func validateState(c echo.Context) error {
	cookieState, err := c.Cookie("state")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "state cookie not found")
	}

	queryState := c.QueryParam("state")

	if cookieState.Value != queryState {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid state")
	}

	c.SetCookie(&http.Cookie{
		Name:     "state",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}
