package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	GithubID  int       `json:"github_id"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`

	EncryptedToken []byte `json:"-"`
}
