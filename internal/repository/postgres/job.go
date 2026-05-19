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
func (j *jobPostgresRepo) UpdateState(ctx context.Context, id uuid.UUID, status model.Status, logURL *string, finishedAt *time.Time) error {
	query := `
		UPDATE jobs
		SET status = $1, finished_at = $2, log_url = $3
		WHERE id = $4;
	`

	tag, err := j.pool.Exec(
		ctx,
		query,
		status,
		finishedAt,
		logURL,
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

// Delete удаляет запись о джобе по её идентификатору.
func (j *jobPostgresRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := j.pool.Exec(ctx, `DELETE FROM jobs WHERE id = $1`, id)

	if err != nil {
		return fmt.Errorf("delete job: %w", err)
	}

	return nil
}

// FailStaleRunning переводит в failed все running-джобы старше olderThan.
// Возвращает ID обновлённых джоб -> нужны для публикации уведомлений.
func (j *jobPostgresRepo) FailStaleRunning(ctx context.Context, olderThan time.Duration) ([]uuid.UUID, error) {
	query := `
		UPDATE jobs
		SET status = 'failed', finished_at = NOW()
		WHERE status = 'running' AND created_at < $1
		RETURNING id;
	`

	rows, err := j.pool.Query(ctx, query, time.Now().Add(-olderThan))
	if err != nil {
		return nil, fmt.Errorf("fail stale running jobs: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan stale job id: %w", err)
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// GetByIDWithOwner возвращает джобу и user_id владельца проекта одним JOIN-запросом.
func (j *jobPostgresRepo) GetByIDWithOwner(ctx context.Context, id uuid.UUID) (*model.Job, uuid.UUID, error) {
	query := `
		SELECT j.id, j.project_id, j.status, j.commit_sha, j.commit_msg, j.log_url, j.created_at, j.finished_at,
		       p.user_id
		FROM jobs j
		JOIN projects p ON p.id = j.project_id
		WHERE j.id = $1;
	`

	var job model.Job
	var logURL *string
	var ownerID uuid.UUID

	if err := j.pool.QueryRow(ctx, query, id).Scan(
		&job.ID,
		&job.ProjectID,
		&job.Status,
		&job.CommitSHA,
		&job.CommitMsg,
		&logURL,
		&job.CreatedAt,
		&job.FinishedAt,
		&ownerID,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, uuid.Nil, ErrJobNotFound
		}
		return nil, uuid.Nil, fmt.Errorf("get job with owner: %w", err)
	}

	if logURL != nil {
		job.LogURL = *logURL
	}

	return &job, ownerID, nil
}

// ListByProjectIDWithOwner возвращает джобы и user_id владельца проекта одним LEFT JOIN-запросом.
// LEFT JOIN от projects гарантирует хотя бы одну строку если проект существует - это позволяет отличить ErrProjectNotFound от пустого списка джоб.
func (j *jobPostgresRepo) ListByProjectIDWithOwner(ctx context.Context, projectID uuid.UUID) ([]*model.Job, uuid.UUID, error) {
	query := `
		SELECT p.user_id,
		       j.id, j.project_id, j.status, j.commit_sha, j.commit_msg, j.log_url, j.created_at, j.finished_at
		FROM projects p
		LEFT JOIN jobs j ON j.project_id = p.id
		WHERE p.id = $1
		ORDER BY j.created_at DESC;
	`

	rows, err := j.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("list jobs by project with owner: %w", err)
	}
	defer rows.Close()

	var ownerID uuid.UUID
	found := false
	jobs := make([]*model.Job, 0)

	for rows.Next() {
		// поля джобы nullable — LEFT JOIN даёт NULL-строку когда джоб нет
		var jID *uuid.UUID
		var jProjectID *uuid.UUID
		var jStatus *string
		var jCommitSHA, jCommitMsg *string
		var jLogURL *string
		var jCreatedAt *time.Time
		var jFinishedAt *time.Time

		if err := rows.Scan(
			&ownerID,
			&jID, &jProjectID, &jStatus, &jCommitSHA, &jCommitMsg, &jLogURL, &jCreatedAt, &jFinishedAt,
		); err != nil {
			return nil, uuid.Nil, fmt.Errorf("scan job with owner: %w", err)
		}
		found = true

		if jID == nil {
			continue // проект есть, но джоб нет
		}

		job := &model.Job{
			ID:         *jID,
			ProjectID:  *jProjectID,
			Status:     model.Status(*jStatus),
			CommitSHA:  *jCommitSHA,
			CommitMsg:  *jCommitMsg,
			CreatedAt:  *jCreatedAt,
			FinishedAt: jFinishedAt,
		}
		if jLogURL != nil {
			job.LogURL = *jLogURL
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, uuid.Nil, fmt.Errorf("rows iteration: %w", err)
	}

	if !found {
		return nil, uuid.Nil, ErrProjectNotFound
	}

	return jobs, ownerID, nil
}
