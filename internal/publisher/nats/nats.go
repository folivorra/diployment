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
	jsonEvent, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event to json: %w", err)
	}

	_, err = n.js.Publish(ctx, nats.SubjectBuildTriggered, jsonEvent)
	if err != nil {
		return fmt.Errorf("publish build event: %w", err)
	}

	return nil
}
