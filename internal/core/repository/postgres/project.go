package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/internal/postgres"

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

func (p *projectPostgresRepo) Create(ctx context.Context, project *model.Project) error {
	query := `
		INSERT INTO projects (user_id, name, repo_url, build_command)
		VALUES ($1, $2, $3, $4);
	`

	_, err := p.pool.Exec(
		ctx,
		query,
		project.UserID,
		project.Name,
		project.RepoURL,
		project.BuildCommand,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return postgres.ErrProjectAlreadyExist
			}
		}

		return fmt.Errorf("create project in db: %w", err)
	}

	return nil
}

func (p *projectPostgresRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	query := `
		SELECT id, user_id, name, repo_url, build_command, created_at
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
		&project.Name,
		&project.RepoURL,
		&project.BuildCommand,
		&project.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrProjectNotFound
		}
		return nil, fmt.Errorf("get project by id from db: %w", err)
	}

	return &project, nil
}
