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
	StreamLogs   = "LOGS"

	SubjectBuildTriggered = "builds.triggered"
	SubjectJobDispatch    = "jobs.dispatch"
	SubjectJobStarted     = "jobs.started"
	SubjectJobFinished    = "jobs.finished"
	SubjectLogsLine       = "logs.line"
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
			Name:      StreamLogs,
			Subjects:  []string{SubjectLogsLine},
			Retention: jetstream.LimitsPolicy, // логи не удаляются после прочтения
			Storage:   jetstream.FileStorage,
			MaxAge:    7 * 24 * time.Hour,
			MaxMsgs:   1_000_000,
		},
	}

	for _, cfg := range streams {
		if _, err := js.CreateOrUpdateStream(ctx, cfg); err != nil {
			return fmt.Errorf("setup stream %s: %w", cfg.Name, err)
		}
	}

	return nil
}
