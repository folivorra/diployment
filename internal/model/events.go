package model

import (
	"github.com/google/uuid"
)

type Status string

const (
	StatusSuccess   Status = "success"
	StatusFailed    Status = "failed"
	StatusPending   Status = "pending"
	StatusBuilding  Status = "building"
	StatusDeploying Status = "deploying"
)

func (s Status) IsValid() bool {
	return s == StatusSuccess || s == StatusFailed || s == StatusPending || s == StatusBuilding || s == StatusDeploying
}

// StaleJob результат FailStale: id джобы и фаза в которой она зависла.
type StaleJob struct {
	ID    uuid.UUID
	Phase Phase
}

// Phase фаза джобы: build или deploy.
type Phase string

const (
	PhaseBuild  Phase = "build"
	PhaseDeploy Phase = "deploy"
)

func (p Phase) IsValid() bool {
	return p == "build" || p == "deploy"
}

// BuildDispatchEvent builds.dispatch (coordinator → builder)
type BuildDispatchEvent struct {
	JobID          uuid.UUID
	ProjectID      uuid.UUID
	CloneURL       string
	Branch         string
	EncryptedToken []byte
}

// JobTriggeredEvent jobs.triggered (webhook → coordinator)
type JobTriggeredEvent struct {
	ProjectID      uuid.UUID
	RepoFullName   string
	CloneURL       string
	Branch         string
	CommitSHA      string
	CommitMsg      string
	EncryptedToken []byte // provider access token
}

// BuildStartedEvent builds.started (builder → coordinator)
type BuildStartedEvent struct {
	JobID     uuid.UUID
	ProjectID uuid.UUID
	WorkerID  string
}

// BuildFinishedEvent builds.finished (builder → coordinator)
type BuildFinishedEvent struct {
	JobID     uuid.UUID
	ProjectID uuid.UUID
	Status    Status // success, failed
	LogURL    string // logs/{job_id}-build.log
	Error     string // if Status == success → error == ""
}

// JobNotifyEvent jobs.notify.{job_id} (coordinator → core)
type JobNotifyEvent struct {
	JobID  uuid.UUID
	Status Status // pending, building, deploying, success, failed
	Phase  Phase  // "build" | "deploy"
	Error  string // if Status != failed → error == ""
}

// JobLogLine jobs.logs.{job_id} (builder/deployer → core)
type JobLogLine struct {
	JobID uuid.UUID
	Line  string
	Phase Phase // "build" | "deploy"
	Done  bool  // true = больше строк не будет для этой фазы
}

// DeployDispatchEvent deploys.dispatch (coordinator → deployer)
type DeployDispatchEvent struct {
	JobID            uuid.UUID
	ProjectID        uuid.UUID
	ImageArtifactKey string // artifacts/{job_id}.tar
	SSHHost          string
	SSHPort          int
	SSHUser          string
	EncryptedSSHKey  []byte // зашифрованный PEM-ключ
	RestartCmd       string
	Workdir          string
}

// DeployStartedEvent deploys.started (deployer → coordinator)
type DeployStartedEvent struct {
	JobID     uuid.UUID
	ProjectID uuid.UUID
	WorkerID  string
}

// DeployFinishedEvent deploys.finished (deployer → coordinator)
type DeployFinishedEvent struct {
	JobID     uuid.UUID
	ProjectID uuid.UUID
	Status    Status // success, failed
	LogURL    string // logs/{job_id}-deploy.log
	Error     string // if Status == success → error == ""
}
