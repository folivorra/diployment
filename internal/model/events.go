package model

import (
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
	ProjectID      uuid.UUID
	RepoFullName   string
	CloneURL       string
	Branch         string
	CommitSHA      string
	CommitMsg      string
	EncryptedToken []byte // provider access token
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
	LogURL    string // ключ объекта в S3, например "{job_id}.log"
	Error     string // if Status == success → error == ""
}

// JobNotifyEvent - job.notify.{job_id} (coordinator → core, статус сборки)
type JobNotifyEvent struct {
	JobID  uuid.UUID
	Status Status // success, failed, pending, running
	Error  string // if Status != failed → error == ""
}

// JobLogLine - jobs.logs.{job_id} (worker → core, стриминг логов сборки)
type JobLogLine struct {
	JobID uuid.UUID
	Line  string
	Done  bool // true = больше строк не будет
}
