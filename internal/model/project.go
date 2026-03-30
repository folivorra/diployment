package model

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Name         string    `json:"name"`
	RepoURL      string    `json:"repo_url"`
	BuildCommand string    `json:"build_command"`
	CreatedAt    time.Time `json:"created_at"`
}
