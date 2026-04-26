package model

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID            uuid.UUID `json:"id"`
	UserID        uuid.UUID `json:"user_id"`
	RepoFullName  string    `json:"repo_full_name"`
	CloneURL      string    `json:"clone_url"`
	WebhookID     int       `json:"webhook_id"`
	WebhookSecret []byte    `json:"-"`
	CreatedAt     time.Time `json:"created_at"`
}
