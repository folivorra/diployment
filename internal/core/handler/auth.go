package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/labstack/echo/v4"
)

type AuthService interface {
	GetAuthCodeURL(state string) string
	Authenticate(ctx context.Context, code string) (string, error)
}

type AuthHandler struct {
	service AuthService
}

func NewAuthHandler(authSrv AuthService) *AuthHandler {
	return &AuthHandler{
		service: authSrv,
	}
}

// Login выполняет редирект на GH для запроса прав.
func (a *AuthHandler) Login(c echo.Context) error {
	state := generateState()

	c.SetCookie(&http.Cookie{
		Name:   "state",
		Value:  state,
		MaxAge: 3600,
		Path:   "/",
		//Domain:   "localhost",
		Secure:   false, // fixme на проде это ДОЛЖНО быть true
		HttpOnly: true,
	})

	url := a.service.GetAuthCodeURL(state)

	return c.Redirect(http.StatusTemporaryRedirect, url)
}

// Callback
func (a *AuthHandler) Callback(c echo.Context) error {
	if err := validateState(c); err != nil {
		return err
	}

	code := c.QueryParam("code")
	if code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "code is missing")
	}

	jwtToken, err := a.service.Authenticate(c.Request().Context(), code)
	if err != nil {
		// todo разделить ошибки (сейчас на все отдаю 500)
		// todo добавить 403 на ошибки обмена токена и 502 на ошибки GH APIU
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	c.SetCookie(&http.Cookie{
		Name:     "jwt_token",
		Value:    jwtToken,
		HttpOnly: true,
		Secure:   false, // fixme на проде это ДОЛЖНО быть true
		Path:     "/",
	})

	return c.Redirect(http.StatusTemporaryRedirect, "http://localhost:3000/dashboard") // fixme добавить фронт кфг
}

// generateState возвращает рандомно сгенерированную строку в base64 для использования как state.
func generateState() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// validateState валидирует state, сравнивая переданное в cookie и в query.
func validateState(c echo.Context) error {
	// достаем state из куки
	cookieState, err := c.Cookie("state")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "state cookie not found")
	}

	// достаем state из query
	queryState := c.QueryParam("state")

	// state`ы должны совпасть - иначе угроза CSRF атаки
	if cookieState.Value != queryState {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid state")
	}

	// удаление куки, чтобы не висела MaxAge
	cookieState.MaxAge = -1
	c.SetCookie(cookieState)

	return nil
}
