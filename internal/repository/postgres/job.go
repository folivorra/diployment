package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/folivorra/diployment/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type jobPostgresRepo struct {
	pool *pgxpool.Pool
}

func NewJobPostgresRepo(pool *pgxpool.Pool) *jobPostgresRepo {
	return &jobPostgresRepo{pool: pool}
}

// Create создает запись о сформированной джобе.
func (j *jobPostgresRepo) Create(ctx context.Context, job *model.Job) (uuid.UUID, error) {
	query := `
		INSERT INTO jobs (project_id, commit_sha, commit_msg)
		VALUES ($1, $2, $3)
		RETURNING id;
	`

	var id uuid.UUID
	err := j.pool.QueryRow(
		ctx,
		query,
		job.ProjectID,
		job.CommitSHA,
		job.CommitMsg,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create job: %w", err)
	}

	return id, nil
}

// UpdateState обновляет состояние в записи о джобе.
func (j *jobPostgresRepo) UpdateState(ctx context.Context, id uuid.UUID, status model.Status, finishedAt *time.Time) error {
	query := `
		UPDATE jobs
		SET status = $1, finished_at = $2
		WHERE id = $3;
	`

	tag, err := j.pool.Exec(
		ctx,
		query,
		status,
		finishedAt,
		id,
	)
	if err != nil {
		return fmt.Errorf("update job status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrJobNotFound
	}

	return nil
}
