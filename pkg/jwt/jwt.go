package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// GenerateAccessToken генерирует новый JWT токен (hmac-sha256)
func GenerateAccessToken(userID uuid.UUID, secret []byte, ttl time.Duration) (string, time.Time, error) {
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
		return "", time.Time{}, err
	}

	return signedToken, exp, nil
}
