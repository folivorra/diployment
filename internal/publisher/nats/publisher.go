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

func (n *natsPublisher) PublishBuildEvent(ctx context.Context, event model.BuildEvent) error {
	return n.publish(ctx, nats.SubjectBuildTriggered, event)
}

func (n *natsPublisher) PublishJobDispatch(ctx context.Context, job model.Job) error {
	return n.publish(ctx, nats.SubjectJobDispatch, job)
}

func (n *natsPublisher) publish(ctx context.Context, subject string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event to json: %w", err)
	}
	_, err = n.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("publish to %s: %w", subject, err)
	}
	return nil
}
