package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/internal/nats"

	natsio "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	CoordinatorBuildsDurable = "coordinator-builds"
	CoordinatorJobsDurable   = "coordinator-jobs"

	FetchTimeout = 2 * time.Second
)

var ErrUnmarshaling = errors.New("unmarshaling failed")

type MsgHandler interface {
	HandleBuildTriggered(ctx context.Context, event model.BuildEvent) error
	HandleJobStarted(ctx context.Context, event model.JobStartedEvent) error
	HandleJobFinished(ctx context.Context, event model.JobFinishedEvent) error
}

type natsConsumer struct {
	js jetstream.JetStream
	eh MsgHandler
}

func NewNatsConsumer(js jetstream.JetStream, mh MsgHandler) *natsConsumer {
	return &natsConsumer{js: js, eh: mh}
}

func (c *natsConsumer) StartConsumeBuilds(ctx context.Context) error {
	return c.start(ctx, nats.StreamBuilds, CoordinatorBuildsDurable, func(msg jetstream.Msg) error {
		event, err := unmarshalMsgIntoEvent[model.BuildEvent](msg.Data())
		if err != nil {
			return err
		}
		return c.eh.HandleBuildTriggered(ctx, event)
	})
}

func (c *natsConsumer) StartConsumeJobs(ctx context.Context) error {
	return c.start(ctx, nats.StreamJobs, CoordinatorJobsDurable, func(msg jetstream.Msg) error {
		// определяем тип события по subject
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
	})
}

// start создает пулл-консюмер из stream`а и тянет сообщения для последующей обработки.
func (c *natsConsumer) start(ctx context.Context, stream string, durable string, handle func(jetstream.Msg) error) error {
	consumer, err := c.js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Durable:   durable,
		AckPolicy: jetstream.AckExplicitPolicy,
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

			if err := handle(msg); err != nil {
				if errors.Is(err, ErrUnmarshaling) {
					slog.Error("unmarshal msg into event", slog.Any("error", err))
				} else {
					slog.Error("handle event failed",
						slog.String("stream", stream),
						slog.Any("error", err),
					)
				}
				_ = msg.Nak()
				continue
			}

			_ = msg.Ack()
		}
	}

}

// unmarshalMsgIntoEvent десериализует сообщение в тип события(T).
func unmarshalMsgIntoEvent[T any](rawMsg []byte) (T, error) {
	var event T
	if err := json.Unmarshal(rawMsg, &event); err != nil {
		return event, fmt.Errorf("%w: %w", ErrUnmarshaling, err)
	}
	return event, nil
}
