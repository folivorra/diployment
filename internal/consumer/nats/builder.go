package nats

import (
	"context"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/internal/nats"
	"github.com/folivorra/diployment/pkg/decode"

	"github.com/nats-io/nats.go/jetstream"
)

// BuildExecutor обрабатывает событие builds.dispatch и запускает сборку.
type BuildExecutor interface {
	Execute(ctx context.Context, event model.BuildDispatchEvent) error
}

type natsBuilderConsumer struct {
	js       jetstream.JetStream
	executor BuildExecutor
}

// NewNatsBuildConsumer создаёт консюмер для билдера.
func NewNatsBuildConsumer(js jetstream.JetStream, executor BuildExecutor) *natsBuilderConsumer {
	return &natsBuilderConsumer{js: js, executor: executor}
}

// StartConsumeBuildsDispatch подписывается на builds.dispatch (coordinator → builder).
func (c *natsBuilderConsumer) StartConsumeBuildsDispatch(ctx context.Context) error {
	return consume(ctx, c.js, nats.StreamBuilds,
		jetstream.ConsumerConfig{
			Durable:       DurableBuilders,
			FilterSubject: nats.SubjectBuildsDispatch,
			AckPolicy:     jetstream.AckExplicitPolicy,
			MaxDeliver:    MaxDeliverAttempts,
		},
		func(msg jetstream.Msg) error {
			event, err := decode.UnmarshalMsgIntoEvent[model.BuildDispatchEvent](msg.Data())
			if err != nil {
				return err
			}
			return c.executor.Execute(ctx, event)
		},
	)
}
