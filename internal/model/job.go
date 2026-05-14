package model

import (
	"time"

	"github.com/google/uuid"
)

type Job struct {
	ID             uuid.UUID
	ProjectID      uuid.UUID
	Status         Status // pending, running, success, failed
	Branch         string
	CloneURL       string
	EncryptedToken []byte // provider access token
	CommitSHA      string
	CommitMsg      string
	LogURL         string
	CreatedAt      time.Time
	FinishedAt     *time.Time
}
