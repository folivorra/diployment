package service

import (
	"context"
	"fmt"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/pkg/crypto/aesgcm"
	"github.com/google/uuid"
)

type UserGetter interface {
	GetByID(ctx context.Context, id *uuid.UUID) (*model.User, error)
}

type RepoLister interface {
	ListUserRepos(ctx context.Context, ownerToken string) ([]*model.Repository, error)
}

type repoService struct {
	lister RepoLister
	getter UserGetter
	key    string
}

func NewRepoService(lister RepoLister, getter UserGetter, key string) *repoService {
	return &repoService{lister: lister, getter: getter, key: key}
}

func (r *repoService) ListUserReposByID(ctx context.Context, userID *uuid.UUID) ([]*model.Repository, error) {
	user, err := r.getter.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	token, err := aesgcm.Decrypt(user.EncryptedToken, r.key, []byte("USER-DATA")) // fixme константная user-data
	if err != nil {
		return nil, fmt.Errorf("decrypt user token: %w", err)
	}

	repos, err := r.lister.ListUserRepos(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProviderAPI, err)
	}

	return repos, nil
}
