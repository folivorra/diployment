package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/pkg/crypto/aesgcm"

	"github.com/google/uuid"
)

type ProjectCreater interface {
	Create(ctx context.Context, project *model.Project) error
}

type ProjectLister interface {
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Project, error)
}

type WebhookManager interface {
	CreateWebhook(ctx context.Context, token string, repoFullName string, webhookSecret string) (int, error)
	DeleteWebhook(ctx context.Context, token string, repoFullName string, webhookID int) error
}

type RepoOwnerGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type ImportProjectInput struct {
	RepoFullName string `json:"repo_full_name"`
	CloneURL     string `json:"clone_url"`
	Branch       string `json:"branch"`
}

func (i ImportProjectInput) IsValid() bool {
	return i.RepoFullName != "" && i.CloneURL != "" && i.Branch != ""
}

type projectService struct {
	pc  ProjectCreater
	pl  ProjectLister
	wm  WebhookManager
	og  RepoOwnerGetter
	key []byte
}

func NewProjectService(pc ProjectCreater, pl ProjectLister, wm WebhookManager, og RepoOwnerGetter, key []byte) *projectService {
	return &projectService{pc: pc, pl: pl, wm: wm, og: og, key: key}
}

func (p *projectService) List(ctx context.Context, userID uuid.UUID) ([]*model.Project, error) {
	return p.pl.ListByUserID(ctx, userID)
}

func (p *projectService) Import(ctx context.Context, ownerID uuid.UUID, input ImportProjectInput) error {
	owner, err := p.og.GetByID(ctx, ownerID)
	if err != nil {
		return fmt.Errorf("get repo owner: %w", err)
	}

	decryptedToken, err := aesgcm.Decrypt(owner.EncryptedToken, p.key, aesgcm.GitHubToken)
	if err != nil {
		return fmt.Errorf("decrypt repo owner token: %w", err)
	}

	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return err
	}
	webhookSecret := base64.StdEncoding.EncodeToString(secret)

	wID, err := p.wm.CreateWebhook(ctx, string(decryptedToken), input.RepoFullName, webhookSecret)
	if err != nil {
		return fmt.Errorf("%w: create webhook on repo: %w", ErrProviderAPI, err)
	}

	encryptedWebhookSecret, err := aesgcm.Encrypt(webhookSecret, p.key, aesgcm.WebhookSecret)
	if err != nil {
		_ = p.wm.DeleteWebhook(ctx, string(decryptedToken), input.RepoFullName, wID)
		return fmt.Errorf("encrypt webhook secret: %w", err)
	}

	project := &model.Project{
		UserID:        owner.ID,
		RepoFullName:  input.RepoFullName,
		Branch:        input.Branch,
		CloneURL:      input.CloneURL,
		WebhookID:     wID,
		WebhookSecret: encryptedWebhookSecret,
	}
	if err = p.pc.Create(ctx, project); err != nil {
		_ = p.wm.DeleteWebhook(ctx, string(decryptedToken), input.RepoFullName, wID)
		return fmt.Errorf("create project: %w", err)
	}

	return nil
}
