package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/internal/nats"
	"github.com/folivorra/diployment/pkg/decode"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	ChanCapacity    = 10
	ChanSendTimeout = 30 * time.Second
)

type subscriberNats struct {
	js jetstream.JetStream
}

func NewSubscriberNats(js jetstream.JetStream) *subscriberNats {
	return &subscriberNats{js: js}
}

func (s *subscriberNats) SubscribeNotifications(ctx context.Context, jobID uuid.UUID) (<-chan model.JobNotifyEvent, func(), error) {
	return subscribe[model.JobNotifyEvent](ctx, s.js, nats.StreamNotifications, jetstream.ConsumerConfig{
		FilterSubject: nats.JobNotifySubject(jobID),
		DeliverPolicy: jetstream.DeliverNewPolicy,
		AckPolicy:     jetstream.AckNonePolicy,
	})
}

func (s *subscriberNats) SubscribeLogs(ctx context.Context, jobID uuid.UUID) (<-chan model.JobLogLine, func(), error) {
	return subscribe[model.JobLogLine](ctx, s.js, nats.StreamLogs, jetstream.ConsumerConfig{
		FilterSubject: nats.JobLogSubject(jobID),
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckNonePolicy,
	})
}

func subscribe[T any](ctx context.Context, js jetstream.JetStream, stream string, cfg jetstream.ConsumerConfig) (<-chan T, func(), error) {
	cons, err := js.CreateOrUpdateConsumer(ctx, stream, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("create consumer: %w", err)
	}

	ch := make(chan T, ChanCapacity)

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		event, err := decode.UnmarshalMsgIntoEvent[T](msg.Data())
		if err != nil {
			return
		}
		select {
		case <-ctx.Done():
		case <-time.After(ChanSendTimeout):
		case ch <- event:
		}
	})
	if err != nil {
		return nil, nil, fmt.Errorf("consume msg: %w", err)
	}

	stop := func() {
		cc.Stop()
		close(ch)
	}

	return ch, stop, nil
}