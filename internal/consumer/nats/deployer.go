package nats

import (
	"context"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/internal/nats"
	"github.com/folivorra/diployment/pkg/decode"

	"github.com/nats-io/nats.go/jetstream"
)

// DeployExecutor обрабатывает событие deploys.dispatch и запускает деплой.
type DeployExecutor interface {
	Execute(ctx context.Context, event model.DeployDispatchEvent) error
}

type natsDeployerConsumer struct {
	js       jetstream.JetStream
	executor DeployExecutor
}

// NewNatsDeployerConsumer создаёт консюмер для деплоера.
func NewNatsDeployerConsumer(js jetstream.JetStream, executor DeployExecutor) *natsDeployerConsumer {
	return &natsDeployerConsumer{js: js, executor: executor}
}

// StartConsumeDeployDispatch подписывается на deploys.dispatch (coordinator → deployer).
func (c *natsDeployerConsumer) StartConsumeDeployDispatch(ctx context.Context) error {
	return consume(ctx, c.js, nats.StreamDeploys,
		jetstream.ConsumerConfig{
			Durable:       DurableDeployers,
			FilterSubject: nats.SubjectDeploysDispatch,
			AckPolicy:     jetstream.AckExplicitPolicy,
			MaxDeliver:    MaxDeliverAttempts,
		},
		func(msg jetstream.Msg) error {
			event, err := decode.UnmarshalMsgIntoEvent[model.DeployDispatchEvent](msg.Data())
			if err != nil {
				return err
			}
			return c.executor.Execute(ctx, event)
		},
	)
}
