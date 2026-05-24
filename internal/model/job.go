package model

import (
	"time"

	"github.com/google/uuid"
)

type Job struct {
	ID               uuid.UUID
	ProjectID        uuid.UUID
	Status           Status // pending, building, deploying, success, failed
	Branch           string
	CloneURL         string
	EncryptedToken   []byte // provider access token
	CommitSHA        string
	CommitMsg        string
	CreatedAt        time.Time
	BuildLogURL      string
	BuildStartedAt   *time.Time
	BuildFinishedAt  *time.Time
	DeployLogURL     string
	DeployStartedAt  *time.Time
	DeployFinishedAt *time.Time
}