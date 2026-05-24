package nats

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	StreamJobs          = "JOBS"
	StreamBuilds        = "BUILDS"
	StreamDeploys       = "DEPLOYS"
	StreamNotifications = "NOTIFICATIONS"
	StreamLogs          = "LOGS"

	SubjectJobTriggered    = "jobs.triggered"
	SubjectBuildsDispatch  = "builds.dispatch"
	SubjectBuildsStarted   = "builds.started"
	SubjectBuildsFinished  = "builds.finished"
	SubjectDeploysDispatch = "deploys.dispatch"
	SubjectDeploysStarted  = "deploys.started"
	SubjectDeploysFinished = "deploys.finished"
	SubjectJobsNotify      = "jobs.notify.*" // job.notify.{job_id}
	SubjectJobsLogs        = "jobs.logs.*"   // jobs.logs.{job_id}
)

// SetupStreams инициализирует streams (topics) в NATS.
func SetupStreams(ctx context.Context, js jetstream.JetStream) error {
	streams := []jetstream.StreamConfig{
		{
			Name:      StreamJobs,
			Subjects:  []string{SubjectJobTriggered},
			Retention: jetstream.WorkQueuePolicy,
			Storage:   jetstream.FileStorage,
			MaxAge:    24 * time.Hour,
		},
		{
			Name:      StreamBuilds,
			Subjects:  []string{SubjectBuildsDispatch, SubjectBuildsStarted, SubjectBuildsFinished},
			Retention: jetstream.WorkQueuePolicy,
			Storage:   jetstream.FileStorage,
			MaxAge:    24 * time.Hour,
		},
		{
			Name:      StreamDeploys,
			Subjects:  []string{SubjectDeploysDispatch, SubjectDeploysStarted, SubjectDeploysFinished},
			Retention: jetstream.WorkQueuePolicy,
			Storage:   jetstream.FileStorage,
			MaxAge:    24 * time.Hour,
		},
		{
			Name:      StreamNotifications,
			Subjects:  []string{SubjectJobsNotify},
			Retention: jetstream.InterestPolicy,
			Storage:   jetstream.MemoryStorage,
			MaxAge:    5 * time.Minute,
		},
		{
			Name:              StreamLogs,
			Subjects:          []string{SubjectJobsLogs},
			Retention:         jetstream.LimitsPolicy,
			Storage:           jetstream.MemoryStorage,
			MaxAge:            2 * time.Hour,
			MaxMsgsPerSubject: 50000,
		},
	}

	for _, cfg := range streams {
		if _, err := js.CreateOrUpdateStream(ctx, cfg); err != nil {
			return fmt.Errorf("setup stream %s: %w", cfg.Name, err)
		}
	}

	return nil
}

func JobNotifySubject(jobID uuid.UUID) string {
	return strings.Replace(SubjectJobsNotify, "*", jobID.String(), 1)
}

func JobLogSubject(jobID uuid.UUID) string {
	return strings.Replace(SubjectJobsLogs, "*", jobID.String(), 1)
}
