package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	StreamBuilds = "BUILDS"
	StreamJobs   = "JOBS"

	SubjectBuildTriggered = "builds.triggered"
	SubjectJobDispatch    = "jobs.dispatch"
	SubjectJobStarted     = "jobs.started"
	SubjectJobFinished    = "jobs.finished"
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
	}

	for _, cfg := range streams {
		if _, err := js.CreateOrUpdateStream(ctx, cfg); err != nil {
			return fmt.Errorf("setup stream %s: %w", cfg.Name, err)
		}
	}

	return nil
}
