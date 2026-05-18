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

func (s *subscriberNats) Subscribe(ctx context.Context, jobID uuid.UUID) (<-chan model.JobNotifyEvent, func(), error) {
	cons, err := s.js.CreateOrUpdateConsumer(ctx, nats.StreamNotifications, jetstream.ConsumerConfig{
		FilterSubject: nats.JobNotifySubject(jobID),
		DeliverPolicy: jetstream.DeliverNewPolicy,
		AckPolicy:     jetstream.AckNonePolicy,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create consumer: %w", err)
	}

	ch := make(chan model.JobNotifyEvent, ChanCapacity)

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		event, err := decode.UnmarshalMsgIntoEvent[model.JobNotifyEvent](msg.Data())
		if err != nil {
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(ChanSendTimeout):
			return
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
