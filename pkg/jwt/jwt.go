package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// GenerateAccessToken генерирует новый JWT токен (hmac-sha256)
func GenerateAccessToken(userID uuid.UUID, secret []byte, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	exp := now.Add(ttl)

	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),         // sub = user_id
		IssuedAt:  jwt.NewNumericDate(now), // iat
		ExpiresAt: jwt.NewNumericDate(exp), // exp
		Issuer:    "diployment-core",       // iss
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("signing token by secret: %w", err)
	}

	return signedToken, nil
}
