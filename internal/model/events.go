package model

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusSuccess Status = "success"
	StatusFailed  Status = "failed"
	StatusPending Status = "pending"
	StatusRunning Status = "running"
)

func (s Status) IsValid() bool {
	return s == "success" || s == "failed" || s == "pending" || s == "running"
}

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
