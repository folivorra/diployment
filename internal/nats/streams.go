package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	StreamNameBuilds = "BUILDS"
	StreamNameJobs   = "JOBS"
	StreamNameLogs   = "LOGS"

	SubjectBuildTriggered = "builds.triggered"
	SubjectJobStarted     = "jobs.started"
	SubjectJobFinished    = "jobs.finished"
	SubjectLogsLine       = "logs.line"
)

func SetupStreams(ctx context.Context, js jetstream.JetStream) error {
	streams := []jetstream.StreamConfig{
		{
			Name:      StreamNameBuilds,
			Subjects:  []string{SubjectBuildTriggered},
			Retention: jetstream.WorkQueuePolicy,
			Storage:   jetstream.FileStorage,
			MaxAge:    24 * time.Hour,
		},
		{
			Name:      StreamNameJobs,
			Subjects:  []string{SubjectJobStarted, SubjectJobFinished},
			Retention: jetstream.WorkQueuePolicy,
			Storage:   jetstream.FileStorage,
			MaxAge:    24 * time.Hour,
		},
		{
			Name:      StreamNameLogs,
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
