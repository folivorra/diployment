package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/folivorra/diployment/pkg/crypto/aesgcm"

	"github.com/folivorra/diployment/internal/config"
	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/pkg/jwt"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

var (
	ErrCodeExchange = errors.New("code exchange failed")
	ErrProviderAPI  = errors.New("provider API error")
)

type UserRepository interface {
	Upsert(ctx context.Context, user *model.User) (*uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type Provider interface {
	AuthCodeURL(state string) string
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*model.User, error)
}

type authService struct {
	provider Provider
	repo     UserRepository
	authCfg  config.AuthConfig
	key      string
}

func NewAuthService(provider Provider, repo UserRepository, authCfg config.AuthConfig, key string) *authService {
	return &authService{
		provider: provider,
		repo:     repo,
		authCfg:  authCfg,
		key:      key,
	}
}

func (a *authService) GetAuthCodeURL(state string) string {
	return a.provider.AuthCodeURL(state)
}

func (a *authService) Authenticate(ctx context.Context, code string) (string, error) {
	token, err := a.provider.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrCodeExchange, err)
	}

	user, err := a.provider.GetUserInfo(ctx, token)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrProviderAPI, err)
	}

	encryptedToken, err := aesgcm.Encrypt(token.AccessToken, a.key, []byte("USER-DATA")) // fixme константная юзер дата
	if err != nil {
		return "", fmt.Errorf("encrypt token: %w", err)
	}

	user.EncryptedToken = encryptedToken

	id, err := a.repo.Upsert(ctx, user)
	if err != nil {
		return "", fmt.Errorf("upsert user: %w", err)
	}

	jwtToken, err := jwt.GenerateAccessToken(id, []byte(a.authCfg.JWTSecret), a.authCfg.JWTTTL)
	if err != nil {
		return "", fmt.Errorf("generate access token: %w", err)
	}

	return jwtToken, nil
}
