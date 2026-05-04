package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/folivorra/diployment/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type projectPostgresRepo struct {
	pool *pgxpool.Pool
}

func NewProjectPostgresRepo(pool *pgxpool.Pool) *projectPostgresRepo {
	return &projectPostgresRepo{
		pool: pool,
	}
}

// Create создает запись о проекте.
func (p *projectPostgresRepo) Create(ctx context.Context, project *model.Project) error {
	query := `
		INSERT INTO projects (user_id, repo_full_name, branch, clone_url, webhook_id, webhook_secret)
		VALUES ($1, $2, $3, $4, $5, $6);
	`

	_, err := p.pool.Exec(
		ctx,
		query,
		project.UserID,
		project.RepoFullName,
		project.Branch,
		project.CloneURL,
		project.WebhookID,
		project.WebhookSecret,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return ErrProjectAlreadyExist
			}
		}
		return fmt.Errorf("create project: %w", err)
	}

	return nil
}

// GetByID возвращает запись о проекте по его идентификатору.
func (p *projectPostgresRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	query := `
		SELECT id, user_id, repo_full_name, branch, clone_url, webhook_id, webhook_secret, created_at
		FROM projects
		WHERE id = $1;
	`

	var project model.Project

	err := p.pool.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&project.ID,
		&project.UserID,
		&project.RepoFullName,
		&project.Branch,
		&project.CloneURL,
		&project.WebhookID,
		&project.WebhookSecret,
		&project.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("get project by id: %w", err)
	}

	return &project, nil
}

// ListByUserID возвращает список проектов по идентификатору пользователя.
func (p *projectPostgresRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Project, error) {
	query := `
		SELECT id, user_id, repo_full_name, branch, clone_url, webhook_id, webhook_secret, created_at
		FROM projects
		WHERE user_id = $1;
	`

	projects := make([]*model.Project, 0)

	r, err := p.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get list of projects by user id: %w", err)
	}
	defer r.Close()

	for r.Next() {
		var project model.Project

		err := r.Scan(
			&project.ID,
			&project.UserID,
			&project.RepoFullName,
			&project.Branch,
			&project.CloneURL,
			&project.WebhookID,
			&project.WebhookSecret,
			&project.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}

		projects = append(projects, &project)
	}
	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return projects, nil
}

// GetByFullName возвращает запись о проекте по полному имени репозитория.
func (p *projectPostgresRepo) GetByFullName(ctx context.Context, fullName string) (*model.Project, error) {
	query := `
		SELECT id, user_id, repo_full_name, branch, clone_url, webhook_id, webhook_secret, created_at
		FROM projects
		WHERE repo_full_name = $1;
	`

	var project model.Project

	err := p.pool.QueryRow(
		ctx,
		query,
		fullName,
	).Scan(
		&project.ID,
		&project.UserID,
		&project.RepoFullName,
		&project.Branch,
		&project.CloneURL,
		&project.WebhookID,
		&project.WebhookSecret,
		&project.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("get project by repo full name: %w", err)
	}

	return &project, nil
}
