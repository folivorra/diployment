package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/internal/nats"
	"github.com/nats-io/nats.go/jetstream"
)

type natsPublisher struct {
	js jetstream.JetStream
}

func NewNatsPublisher(js jetstream.JetStream) *natsPublisher {
	return &natsPublisher{js: js}
}

func (p *natsPublisher) PublishBuildEvent(ctx context.Context, event model.BuildEvent) error {
	return p.publish(ctx, nats.SubjectBuildTriggered, event)
}

func (p *natsPublisher) PublishJobDispatch(ctx context.Context, job model.Job) error {
	return p.publish(ctx, nats.SubjectJobDispatch, job)
}

func (p *natsPublisher) PublishJobStarted(ctx context.Context, event model.JobStartedEvent) error {
	return p.publish(ctx, nats.SubjectJobStarted, event)
}

func (p *natsPublisher) PublishJobFinished(ctx context.Context, event model.JobFinishedEvent) error {
	return p.publish(ctx, nats.SubjectJobFinished, event)
}

func (p *natsPublisher) PublishJobNotify(ctx context.Context, event model.JobNotifyEvent) error {
	return p.publish(ctx, nats.JobNotifySubject(event.JobID), event)
}

func (p *natsPublisher) PublishJobLogLine(ctx context.Context, logLine model.JobLogLine) error {
	return p.publish(ctx, nats.JobLogSubject(logLine.JobID), logLine)
}

func (p *natsPublisher) publish(ctx context.Context, subject string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event to json: %w", err)
	}
	_, err = p.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("publish to %s: %w", subject, err)
	}
	return nil
}
