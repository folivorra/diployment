package nats

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/folivorra/diployment/pkg/decode"
	natsio "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	DurableCoordinatorBuilds  = "coordinator-builds"
	DurableCoordinatorJobs    = "coordinator-jobs"
	DurableCoordinatorDeploys = "coordinator-deploys"
	DurableBuilders  = "builders"
	DurableDeployers = "deployers"

	FetchTimeout       = 2 * time.Second
	MaxDeliverAttempts = 5
)

func consume(
	ctx context.Context,
	js jetstream.JetStream,
	stream string,
	cons jetstream.ConsumerConfig,
	handle func(jetstream.Msg) error,
) error {
	consumer, err := js.CreateOrUpdateConsumer(ctx, stream, cons)
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
				if errors.Is(err, decode.ErrUnmarshaling) {
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
