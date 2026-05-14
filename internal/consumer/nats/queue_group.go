package nats

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/internal/nats"

	natsio "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type JobExecutor interface {
	ExecuteJob(ctx context.Context, job model.Job) error
}

type natsQueueWorkerConsumer struct {
	js       jetstream.JetStream
	executor JobExecutor
}

func NewNatsQueueWorkerConsumer(js jetstream.JetStream, executor JobExecutor) *natsQueueWorkerConsumer {
	return &natsQueueWorkerConsumer{js: js, executor: executor}
}

func (c *natsQueueWorkerConsumer) StartConsumeJobsDispatch(ctx context.Context) error {
	consumer, err := c.js.CreateOrUpdateConsumer(ctx, nats.StreamJobs, jetstream.ConsumerConfig{
		Durable:       DurableWorkers,
		FilterSubject: nats.SubjectJobDispatch,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    MaxDeliverAttempts,
	})
	if err != nil {
		return fmt.Errorf("upsert nats consumer: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			msgs, err := consumer.Fetch(1, jetstream.FetchMaxWait(FetchTimeout))
			if err != nil {
				if !errors.Is(err, natsio.ErrTimeout) {
					slog.Error("fetching fail", slog.Any("error", err))
				}
				continue
			}

			msg, ok := <-msgs.Messages()
			if !ok {
				continue
			}

			job, err := unmarshalMsgIntoEvent[model.Job](msg.Data())
			if err != nil {
				slog.Error("unmarshal msg into event", slog.Any("error", err))
				_ = msg.Nak()
				continue
			}

			if err := c.executor.ExecuteJob(ctx, job); err != nil {
				slog.Error("execute job failed", slog.Any("error", err))
				_ = msg.Nak()
				continue
			}

			_ = msg.Ack()
		}
	}
}
