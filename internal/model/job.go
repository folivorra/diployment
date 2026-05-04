package model

import (
	"time"

	"github.com/google/uuid"
)

type Job struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	Status     Status
	CommitSHA  string
	CommitMsg  string
	LogURL     string
	CreatedAt  time.Time
	FinishedAt *time.Time
}
