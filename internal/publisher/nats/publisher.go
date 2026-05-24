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

func (p *natsPublisher) PublishJobTriggered(ctx context.Context, event model.JobTriggeredEvent) error {
	return p.publish(ctx, nats.SubjectJobTriggered, event)
}

func (p *natsPublisher) PublishBuildsDispatch(ctx context.Context, event model.BuildDispatchEvent) error {
	return p.publish(ctx, nats.SubjectBuildsDispatch, event)
}

func (p *natsPublisher) PublishBuildsStarted(ctx context.Context, event model.BuildStartedEvent) error {
	return p.publish(ctx, nats.SubjectBuildsStarted, event)
}

func (p *natsPublisher) PublishBuildsFinished(ctx context.Context, event model.BuildFinishedEvent) error {
	return p.publish(ctx, nats.SubjectBuildsFinished, event)
}

func (p *natsPublisher) PublishDeployDispatch(ctx context.Context, event model.DeployDispatchEvent) error {
	return p.publish(ctx, nats.SubjectDeploysDispatch, event)
}

func (p *natsPublisher) PublishDeployStarted(ctx context.Context, event model.DeployStartedEvent) error {
	return p.publish(ctx, nats.SubjectDeploysStarted, event)
}

func (p *natsPublisher) PublishDeployFinished(ctx context.Context, event model.DeployFinishedEvent) error {
	return p.publish(ctx, nats.SubjectDeploysFinished, event)
}

func (p *natsPublisher) PublishJobsNotify(ctx context.Context, event model.JobNotifyEvent) error {
	return p.publish(ctx, nats.JobNotifySubject(event.JobID), event)
}

func (p *natsPublisher) PublishJobsLogLine(ctx context.Context, logLine model.JobLogLine) error {
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