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
	StreamBuilds        = "BUILDS"
	StreamJobs          = "JOBS"
	StreamNotifications = "NOTIFICATIONS"

	SubjectBuildTriggered = "builds.triggered"
	SubjectJobDispatch    = "jobs.dispatch"
	SubjectJobStarted     = "jobs.started"
	SubjectJobFinished    = "jobs.finished"
	SubjectJobNotify      = "jobs.notify.*" // job.notify.{job_id}
)

// SetupStreams инициализирует streams (topics) в NATS.
func SetupStreams(ctx context.Context, js jetstream.JetStream) error {
	streams := []jetstream.StreamConfig{
		{
			Name:      StreamBuilds,
			Subjects:  []string{SubjectBuildTriggered},
			Retention: jetstream.WorkQueuePolicy,
			Storage:   jetstream.FileStorage,
			MaxAge:    24 * time.Hour,
		},
		{
			Name:      StreamJobs,
			Subjects:  []string{SubjectJobDispatch, SubjectJobStarted, SubjectJobFinished},
			Retention: jetstream.WorkQueuePolicy,
			Storage:   jetstream.FileStorage,
			MaxAge:    24 * time.Hour,
		},
		{
			Name:      StreamNotifications,
			Subjects:  []string{SubjectJobNotify},
			Retention: jetstream.InterestPolicy,
			Storage:   jetstream.MemoryStorage,
			MaxAge:    5 * time.Minute,
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
	return strings.Replace(SubjectJobNotify, "*", jobID.String(), 1)
}
