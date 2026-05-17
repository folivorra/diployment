package nats

import (
	"context"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/internal/nats"

	"github.com/nats-io/nats.go/jetstream"
)

type MsgHandler interface {
	HandleBuildTriggered(ctx context.Context, event model.BuildEvent) error
	HandleJobStarted(ctx context.Context, event model.JobStartedEvent) error
	HandleJobFinished(ctx context.Context, event model.JobFinishedEvent) error
}

type natsCoordinatorConsumer struct {
	js jetstream.JetStream
	eh MsgHandler
}

func NewNatsCoordinatorConsumer(js jetstream.JetStream, mh MsgHandler) *natsCoordinatorConsumer {
	return &natsCoordinatorConsumer{js: js, eh: mh}
}

func (c *natsCoordinatorConsumer) StartConsumeBuilds(ctx context.Context) error {
	return consume(ctx, c.js, nats.StreamBuilds, DurableCoordinatorBuilds, []string{nats.SubjectBuildTriggered},
		func(msg jetstream.Msg) error {
			event, err := unmarshalMsgIntoEvent[model.BuildEvent](msg.Data())
			if err != nil {
				return err
			}
			return c.eh.HandleBuildTriggered(ctx, event)
		})
}

func (c *natsCoordinatorConsumer) StartConsumeJobsStatus(ctx context.Context) error {
	return consume(ctx, c.js, nats.StreamJobs, DurableCoordinatorJobs, []string{nats.SubjectJobStarted, nats.SubjectJobFinished},
		func(msg jetstream.Msg) error {
			switch msg.Subject() {
			case nats.SubjectJobStarted:
				event, err := unmarshalMsgIntoEvent[model.JobStartedEvent](msg.Data())
				if err != nil {
					return err
				}
				return c.eh.HandleJobStarted(ctx, event)
			case nats.SubjectJobFinished:
				event, err := unmarshalMsgIntoEvent[model.JobFinishedEvent](msg.Data())
				if err != nil {
					return err
				}
				return c.eh.HandleJobFinished(ctx, event)
			}
			return nil
		},
	)
}