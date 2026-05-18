package service

import (
	"context"
	"fmt"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/pkg/crypto/aesgcm"

	"github.com/google/uuid"
)

type UserGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type RepoLister interface {
	ListUserRepos(ctx context.Context, ownerToken string) ([]*model.Repository, error)
	ListRepoBranches(ctx context.Context, token string, repoFullName string) ([]string, error)
}

type repoService struct {
	lister RepoLister
	getter UserGetter
	key    []byte
}

func NewRepoService(lister RepoLister, getter UserGetter, key []byte) *repoService {
	return &repoService{lister: lister, getter: getter, key: key}
}

func (r *repoService) ListUserReposByID(ctx context.Context, userID uuid.UUID) ([]*model.Repository, error) {
	user, err := r.getter.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	token, err := aesgcm.Decrypt(user.EncryptedToken, r.key, []byte("USER-DATA")) // fixme константная user-data
	if err != nil {
		return nil, fmt.Errorf("decrypt user token: %w", err)
	}

	repos, err := r.lister.ListUserRepos(ctx, string(token))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProviderAPI, err)
	}

	return repos, nil
}

func (r *repoService) ListRepoBranchesByID(ctx context.Context, userID uuid.UUID, repoFullName string) ([]string, error) {
	user, err := r.getter.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	token, err := aesgcm.Decrypt(user.EncryptedToken, r.key, []byte("USER-DATA"))
	if err != nil {
		return nil, fmt.Errorf("decrypt user token: %w", err)
	}

	branches, err := r.lister.ListRepoBranches(ctx, string(token), repoFullName)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProviderAPI, err)
	}

	return branches, nil
}
