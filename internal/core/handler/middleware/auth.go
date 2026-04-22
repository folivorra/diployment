package middleware

import (
	"errors"
	"net/http"

	"github.com/folivorra/diployment/pkg/jwt"
	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func AuthMiddleware(jwtSecret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			jwtToken, err := c.Cookie("jwt_token")
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "missing token")
			}

			userID, err := jwt.ParseAccessToken(jwtToken.Value, jwtSecret)
			if err != nil {
				if errors.Is(err, gojwt.ErrTokenExpired) {
					return echo.NewHTTPError(http.StatusUnauthorized, "token expired")
				}
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}

			c.Set("user_id", userID)

			return next(c)
		}
	}
}
