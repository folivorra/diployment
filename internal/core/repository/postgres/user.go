package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/internal/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userPostgresRepo struct {
	pool *pgxpool.Pool
}

func NewUserPostgresRepo(pool *pgxpool.Pool) *userPostgresRepo {
	return &userPostgresRepo{
		pool: pool,
	}
}

// Upsert создает запись о пользователе.
//
// Если запись о пользователе с таким же github_id уже существует - обновляет информацию.
func (u *userPostgresRepo) Upsert(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (github_id, avatar_url, github_token) 
		VALUES ($1, $2, $3)
		ON CONFLICT (github_id) 
		DO UPDATE SET 
			avatar_url = EXCLUDED.avatar_url,
			github_token = EXCLUDED.github_token;
	`

	_, err := u.pool.Exec(
		ctx,
		query,
		user.GithubID,
		user.AvatarURL,
		user.EncryptedToken,
	)

	if err != nil {
		return fmt.Errorf("save user into db: %w", err)
	}

	return nil
}

// GetByID возвращает запись о пользователе по его идентификатору.
func (u *userPostgresRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, github_id, avatar_url, github_token, created_at 
		FROM users 
		WHERE id = $1;
	`

	var user model.User

	err := u.pool.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&user.ID,
		&user.GithubID,
		&user.AvatarURL,
		&user.EncryptedToken,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user from db: %w", err)
	}

	return &user, nil
}
