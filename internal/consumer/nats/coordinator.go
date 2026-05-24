package nats

import (
	"context"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/internal/nats"
	"github.com/folivorra/diployment/pkg/decode"

	"github.com/nats-io/nats.go/jetstream"
)

// MsgHandler бизнес-логика координатора, вызывается из каждого консюмера.
type MsgHandler interface {
	HandleJobTriggered(ctx context.Context, event model.JobTriggeredEvent) error
	HandleBuildStarted(ctx context.Context, event model.BuildStartedEvent) error
	HandleBuildFinished(ctx context.Context, event model.BuildFinishedEvent) error
	HandleDeployStarted(ctx context.Context, event model.DeployStartedEvent) error
	HandleDeployFinished(ctx context.Context, event model.DeployFinishedEvent) error
}

type natsCoordinatorConsumer struct {
	js jetstream.JetStream
	eh MsgHandler
}

// NewNatsCoordinatorConsumer создаёт консюмер координатора.
func NewNatsCoordinatorConsumer(js jetstream.JetStream, mh MsgHandler) *natsCoordinatorConsumer {
	return &natsCoordinatorConsumer{js: js, eh: mh}
}

// StartConsumeJobsTriggered подписывается на jobs.triggered (webhook → coordinator).
func (c *natsCoordinatorConsumer) StartConsumeJobsTriggered(ctx context.Context) error {
	return consume(ctx, c.js, nats.StreamJobs,
		jetstream.ConsumerConfig{
			Durable:       DurableCoordinatorBuilds,
			FilterSubject: nats.SubjectJobTriggered,
			AckPolicy:     jetstream.AckExplicitPolicy,
			MaxDeliver:    MaxDeliverAttempts,
		},
		func(msg jetstream.Msg) error {
			event, err := decode.UnmarshalMsgIntoEvent[model.JobTriggeredEvent](msg.Data())
			if err != nil {
				return err
			}
			return c.eh.HandleJobTriggered(ctx, event)
		},
	)
}

// StartConsumeDeploysStatus подписывается на deploys.started и deploys.finished (deployer → coordinator).
func (c *natsCoordinatorConsumer) StartConsumeDeploysStatus(ctx context.Context) error {
	return consume(ctx, c.js, nats.StreamDeploys,
		jetstream.ConsumerConfig{
			Durable:        DurableCoordinatorDeploys,
			FilterSubjects: []string{nats.SubjectDeploysStarted, nats.SubjectDeploysFinished},
			AckPolicy:      jetstream.AckExplicitPolicy,
			MaxDeliver:     MaxDeliverAttempts,
		},
		func(msg jetstream.Msg) error {
			switch msg.Subject() {
			case nats.SubjectDeploysStarted:
				event, err := decode.UnmarshalMsgIntoEvent[model.DeployStartedEvent](msg.Data())
				if err != nil {
					return err
				}
				return c.eh.HandleDeployStarted(ctx, event)
			case nats.SubjectDeploysFinished:
				event, err := decode.UnmarshalMsgIntoEvent[model.DeployFinishedEvent](msg.Data())
				if err != nil {
					return err
				}
				return c.eh.HandleDeployFinished(ctx, event)
			}
			return nil
		},
	)
}

// StartConsumeBuildsStatus подписывается на builds.started и builds.finished (builder → coordinator).
func (c *natsCoordinatorConsumer) StartConsumeBuildsStatus(ctx context.Context) error {
	return consume(ctx, c.js, nats.StreamBuilds,
		jetstream.ConsumerConfig{
			Durable:        DurableCoordinatorJobs,
			FilterSubjects: []string{nats.SubjectBuildsStarted, nats.SubjectBuildsFinished},
			AckPolicy:      jetstream.AckExplicitPolicy,
			MaxDeliver:     MaxDeliverAttempts,
		},
		func(msg jetstream.Msg) error {
			switch msg.Subject() {
			case nats.SubjectBuildsStarted:
				event, err := decode.UnmarshalMsgIntoEvent[model.BuildStartedEvent](msg.Data())
				if err != nil {
					return err
				}
				return c.eh.HandleBuildStarted(ctx, event)
			case nats.SubjectBuildsFinished:
				event, err := decode.UnmarshalMsgIntoEvent[model.BuildFinishedEvent](msg.Data())
				if err != nil {
					return err
				}
				return c.eh.HandleBuildFinished(ctx, event)
			}
			return nil
		},
	)
}