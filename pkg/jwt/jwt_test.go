package jwt_test

import (
	"testing"
	"time"

	"github.com/folivorra/diployment/pkg/jwt"
	"github.com/google/uuid"
)

func TestGenerateAccessToken(t *testing.T) {
	userID := uuid.New()
	secret := []byte("test-secret")
	ttl := time.Hour

	token, err := jwt.GenerateAccessToken(userID, secret, ttl)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	if token == "" {
		t.Fatal("generated token is empty")
	}
}
