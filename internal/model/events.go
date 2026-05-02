package model

import (
	"time"

	"github.com/google/uuid"
)

// BuildEvent - builds.triggered (webhook → coordinator)
type BuildEvent struct {
	ProjectID    uuid.UUID
	RepoFullName string
	CloneURL     string
	Branch       string
	CommitSHA    string
	CommitMsg    string
}

// JobStartedEvent - jobs.started (worker → coordinator, воркер взял задачу)
type JobStartedEvent struct {
	JobID     uuid.UUID
	ProjectID uuid.UUID
	WorkerID  string
}

type Status string

func (s Status) IsValid() bool {
	return s == "success" || s == "failed"
}

// JobFinishedEvent - jobs.finished (worker → coordinator, сборка завершена)
type JobFinishedEvent struct {
	JobID     uuid.UUID
	ProjectID uuid.UUID
	Status    Status // success, failed
	Error     string // if Status == success → error == ""
}

// LogLineEvent - logs.line (worker → log streamer)
type LogLineEvent struct {
	JobID     uuid.UUID
	Line      string
	Timestamp time.Time
}
