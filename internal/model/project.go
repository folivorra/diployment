package model

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID               uuid.UUID `json:"id"`
	UserID           uuid.UUID `json:"user_id"`
	RepoFullName     string    `json:"repo_full_name"`
	Branch           string    `json:"branch"`
	CloneURL         string    `json:"clone_url"`
	WebhookID        int       `json:"webhook_id"`
	WebhookSecret    []byte    `json:"-"`
	CreatedAt        time.Time `json:"created_at"`
	SSHHost          string    `json:"ssh_host"`
	SSHPort          int       `json:"ssh_port"`
	SSHUser          string    `json:"ssh_user"`
	SSHKey           []byte    `json:"-"`
	DeployRestartCmd string    `json:"deploy_restart_cmd"`
	DeployWorkdir    string    `json:"deploy_workdir"`
}