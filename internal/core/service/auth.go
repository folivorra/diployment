package service

import (
	"context"
	"fmt"

	"github.com/folivorra/diployment/pkg/crypto/aesgcm"

	"github.com/folivorra/diployment/internal/config"
	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/pkg/jwt"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type UserRepository interface {
	Upsert(ctx context.Context, user *model.User) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type Provider interface {
	AuthCodeURL(state string) string
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*model.User, error)
}

type AuthService struct {
	provider Provider
	repo     UserRepository
	authCfg  config.AuthConfig
}

func NewAuthService(provider Provider, repo UserRepository, authCfg config.AuthConfig) *AuthService {
	return &AuthService{
		provider: provider,
		repo:     repo,
		authCfg:  authCfg,
	}
}

func (a *AuthService) GetAuthCodeURL(state string) string {
	return a.provider.AuthCodeURL(state)
}

func (a *AuthService) Authenticate(ctx context.Context, code string) (string, error) {
	token, err := a.provider.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("exchange code: %w", err) // todo 403
	}

	user, err := a.provider.GetUserInfo(ctx, token)
	if err != nil {
		return "", fmt.Errorf("get user info: %w", err) // todo 502 или 500 (decoder err)
	}

	encryptedToken, err := aesgcm.Encrypt(token.AccessToken, a.authCfg.MasterKey, []byte("USER-DATA"))
	if err != nil {
		return "", fmt.Errorf("encrypt token: %w", err) // todo 500
	}

	user.EncryptedToken = encryptedToken

	id, err := a.repo.Upsert(ctx, user)
	if err != nil {
		return "", fmt.Errorf("upsert user info: %w", err) // todo 500
	}

	jwtToken, err := jwt.GenerateAccessToken(id, []byte(a.authCfg.JWTSecret), a.authCfg.JWTTTL)
	if err != nil {
		return "", fmt.Errorf("generate access token: %w", err) // todo 500
	}

	return jwtToken, nil
}
