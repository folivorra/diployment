package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/folivorra/diployment/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

// ListByProjectID возвращает список джоб по идентификатору проекта.
func (j *jobPostgresRepo) ListByProjectID(ctx context.Context, projectID uuid.UUID) ([]*model.Job, error) {
	query := `
		SELECT id, project_id, status, commit_sha, commit_msg, log_url, created_at, finished_at
		FROM jobs
		WHERE project_id = $1
		ORDER BY created_at DESC;
	`

	rows, err := j.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("list jobs by project: %w", err)
	}
	defer rows.Close()

	jobs := make([]*model.Job, 0)
	for rows.Next() {
		var job model.Job
		var logURL *string

		if err := rows.Scan(
			&job.ID,
			&job.ProjectID,
			&job.Status,
			&job.CommitSHA,
			&job.CommitMsg,
			&logURL,
			&job.CreatedAt,
			&job.FinishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}

		if logURL != nil {
			job.LogURL = *logURL
		}
		jobs = append(jobs, &job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return jobs, nil
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

// GetByID возвращает запись о джобе по ее идентификатору.
func (j *jobPostgresRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Job, error) {
	query := `
		SELECT id, project_id, status, commit_sha, commit_msg, log_url, created_at, finished_at
		FROM jobs
		WHERE id = $1;
	`

	var job model.Job
	var logURL *string

	if err := j.pool.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&job.ID,
		&job.ProjectID,
		&job.Status,
		&job.CommitSHA,
		&job.CommitMsg,
		&logURL,
		&job.CreatedAt,
		&job.FinishedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("get job: %w", err)
	}

	if logURL != nil {
		job.LogURL = *logURL
	}

	return &job, nil
}
