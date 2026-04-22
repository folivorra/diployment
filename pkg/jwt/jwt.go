package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// GenerateAccessToken генерирует новый JWT токен (hmac-sha256)
func GenerateAccessToken(userID *uuid.UUID, secret []byte, ttl time.Duration) (string, error) {
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

func ParseAccessToken(token string, secret string) (*uuid.UUID, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	sub, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("parsing claims: %w", err)
	}

	return &sub, nil
}
