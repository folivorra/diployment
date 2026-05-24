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

// NewJobPostgresRepo создаёт репозиторий джоб.
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

// UpdateBuildStarted переводит джобу в статус building и фиксирует время старта сборки.
func (j *jobPostgresRepo) UpdateBuildStarted(ctx context.Context, id uuid.UUID, startedAt time.Time) error {
	query := `UPDATE jobs SET status = 'building', build_started_at = $1 WHERE id = $2;`

	tag, err := j.pool.Exec(ctx, query, startedAt, id)
	if err != nil {
		return fmt.Errorf("update build started: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrJobNotFound
	}
	return nil
}

// UpdateBuildFinished сохраняет итог фазы сборки.
func (j *jobPostgresRepo) UpdateBuildFinished(ctx context.Context, id uuid.UUID, status model.Status, buildLogURL *string, finishedAt time.Time) error {
	query := `UPDATE jobs SET status = $1, build_log_url = $2, build_finished_at = $3 WHERE id = $4;`

	tag, err := j.pool.Exec(ctx, query, status, buildLogURL, finishedAt, id)
	if err != nil {
		return fmt.Errorf("update build finished: %w", err)
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

// FailStale переводит в failed все building/deploying-джобы старше olderThan.
// Возвращает список стейлых джоб с ID и фазой, в которой они зависли.
func (j *jobPostgresRepo) FailStale(ctx context.Context, olderThan time.Duration) ([]model.StaleJob, error) {
	query := `
		WITH stale AS (
			SELECT id,
			       CASE WHEN deploy_started_at IS NOT NULL THEN 'deploy' ELSE 'build' END AS phase
			FROM jobs
			WHERE status IN ('building', 'deploying') AND created_at < $1
		)
		UPDATE jobs
		SET status = 'failed',
		    build_finished_at  = CASE WHEN build_started_at  IS NOT NULL AND build_finished_at  IS NULL THEN NOW() ELSE build_finished_at  END,
		    deploy_finished_at = CASE WHEN deploy_started_at IS NOT NULL AND deploy_finished_at IS NULL THEN NOW() ELSE deploy_finished_at END
		FROM stale
		WHERE jobs.id = stale.id
		RETURNING jobs.id, stale.phase;
	`

	rows, err := j.pool.Query(ctx, query, time.Now().Add(-olderThan))
	if err != nil {
		return nil, fmt.Errorf("fail stale jobs: %w", err)
	}
	defer rows.Close()

	var jobs []model.StaleJob
	for rows.Next() {
		var job model.StaleJob
		var phase string
		if err := rows.Scan(&job.ID, &phase); err != nil {
			return nil, fmt.Errorf("scan stale job: %w", err)
		}
		job.Phase = model.Phase(phase)
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// UpdateDeployStarted переводит джобу в статус deploying и фиксирует время старта деплоя.
func (j *jobPostgresRepo) UpdateDeployStarted(ctx context.Context, id uuid.UUID, startedAt time.Time) error {
	query := `UPDATE jobs SET status = 'deploying', deploy_started_at = $1 WHERE id = $2;`

	tag, err := j.pool.Exec(ctx, query, startedAt, id)
	if err != nil {
		return fmt.Errorf("update deploy started: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrJobNotFound
	}
	return nil
}

// UpdateDeployFinished сохраняет итог фазы деплоя.
func (j *jobPostgresRepo) UpdateDeployFinished(ctx context.Context, id uuid.UUID, status model.Status, deployLogURL *string, finishedAt time.Time) error {
	query := `UPDATE jobs SET status = $1, deploy_log_url = $2, deploy_finished_at = $3 WHERE id = $4;`

	tag, err := j.pool.Exec(ctx, query, status, deployLogURL, finishedAt, id)
	if err != nil {
		return fmt.Errorf("update deploy finished: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrJobNotFound
	}
	return nil
}

// GetByIDWithOwner возвращает джобу и user_id владельца проекта одним JOIN-запросом.
func (j *jobPostgresRepo) GetByIDWithOwner(ctx context.Context, id uuid.UUID) (*model.Job, uuid.UUID, error) {
	query := `
		SELECT j.id, j.project_id, j.status, j.commit_sha, j.commit_msg, j.created_at,
		       j.build_log_url, j.build_started_at, j.build_finished_at,
		       j.deploy_log_url, j.deploy_started_at, j.deploy_finished_at,
		       p.user_id
		FROM jobs j
		JOIN projects p ON p.id = j.project_id
		WHERE j.id = $1;
	`

	var job model.Job
	var buildLogURL, deployLogURL *string
	var ownerID uuid.UUID

	if err := j.pool.QueryRow(ctx, query, id).Scan(
		&job.ID, &job.ProjectID, &job.Status, &job.CommitSHA, &job.CommitMsg, &job.CreatedAt,
		&buildLogURL, &job.BuildStartedAt, &job.BuildFinishedAt,
		&deployLogURL, &job.DeployStartedAt, &job.DeployFinishedAt,
		&ownerID,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, uuid.Nil, ErrJobNotFound
		}
		return nil, uuid.Nil, fmt.Errorf("get job with owner: %w", err)
	}

	if buildLogURL != nil {
		job.BuildLogURL = *buildLogURL
	}
	if deployLogURL != nil {
		job.DeployLogURL = *deployLogURL
	}

	return &job, ownerID, nil
}

// ListByProjectIDWithOwner возвращает джобы и user_id владельца проекта одним LEFT JOIN-запросом.
// LEFT JOIN от projects гарантирует хотя бы одну строку если проект существует — это позволяет отличить ErrProjectNotFound от пустого списка джоб.
func (j *jobPostgresRepo) ListByProjectIDWithOwner(ctx context.Context, projectID uuid.UUID) ([]*model.Job, uuid.UUID, error) {
	query := `
		SELECT p.user_id,
		       j.id, j.project_id, j.status, j.commit_sha, j.commit_msg, j.created_at,
		       j.build_log_url, j.build_started_at, j.build_finished_at,
		       j.deploy_log_url, j.deploy_started_at, j.deploy_finished_at
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
		// поля джобы nullable: LEFT JOIN даёт NULL-строку когда джоб нет
		var jID *uuid.UUID
		var jProjectID *uuid.UUID
		var jStatus *string
		var jCommitSHA, jCommitMsg *string
		var jCreatedAt *time.Time
		var jBuildLogURL, jDeployLogURL *string
		var jBuildStartedAt, jBuildFinishedAt *time.Time
		var jDeployStartedAt, jDeployFinishedAt *time.Time

		if err := rows.Scan(
			&ownerID,
			&jID, &jProjectID, &jStatus, &jCommitSHA, &jCommitMsg, &jCreatedAt,
			&jBuildLogURL, &jBuildStartedAt, &jBuildFinishedAt,
			&jDeployLogURL, &jDeployStartedAt, &jDeployFinishedAt,
		); err != nil {
			return nil, uuid.Nil, fmt.Errorf("scan job with owner: %w", err)
		}
		found = true

		if jID == nil {
			continue // проект есть, но джоб нет
		}

		job := &model.Job{
			ID:               *jID,
			ProjectID:        *jProjectID,
			Status:           model.Status(*jStatus),
			CommitSHA:        *jCommitSHA,
			CommitMsg:        *jCommitMsg,
			CreatedAt:        *jCreatedAt,
			BuildStartedAt:   jBuildStartedAt,
			BuildFinishedAt:  jBuildFinishedAt,
			DeployStartedAt:  jDeployStartedAt,
			DeployFinishedAt: jDeployFinishedAt,
		}
		if jBuildLogURL != nil {
			job.BuildLogURL = *jBuildLogURL
		}
		if jDeployLogURL != nil {
			job.DeployLogURL = *jDeployLogURL
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
