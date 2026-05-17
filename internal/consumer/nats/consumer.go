package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	natsio "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	DurableCoordinatorBuilds = "coordinator-builds"
	DurableCoordinatorJobs   = "coordinator-jobs"
	DurableWorkers           = "workers"

	FetchTimeout       = 2 * time.Second
	MaxDeliverAttempts = 5
)

var ErrUnmarshaling = errors.New("unmarshaling failed")

// unmarshalMsgIntoEvent десериализует сообщение в тип события(T).
func unmarshalMsgIntoEvent[T any](rawMsg []byte) (T, error) {
	var event T
	if err := json.Unmarshal(rawMsg, &event); err != nil {
		return event, fmt.Errorf("%w: %w", ErrUnmarshaling, err)
	}
	return event, nil
}

func consume(ctx context.Context, js jetstream.JetStream, stream, durable string, filterSubs []string, handle func(jetstream.Msg) error) error {
	consumer, err := js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Durable:        durable,
		FilterSubjects: filterSubs,
		AckPolicy:      jetstream.AckExplicitPolicy,
		MaxDeliver:     MaxDeliverAttempts,
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