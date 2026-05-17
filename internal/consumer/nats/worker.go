package nats

import (
	"context"

	"github.com/folivorra/diployment/internal/model"
	"github.com/folivorra/diployment/internal/nats"

	"github.com/nats-io/nats.go/jetstream"
)

type JobExecutor interface {
	ExecuteJob(ctx context.Context, job model.Job) error
}

type natsWorkerConsumer struct {
	js       jetstream.JetStream
	executor JobExecutor
}

func NewNatsWorkerConsumer(js jetstream.JetStream, executor JobExecutor) *natsWorkerConsumer {
	return &natsWorkerConsumer{js: js, executor: executor}
}

func (c *natsWorkerConsumer) StartConsumeJobsDispatch(ctx context.Context) error {
	return consume(ctx, c.js, nats.StreamJobs, DurableWorkers, []string{nats.SubjectJobDispatch},
		func(msg jetstream.Msg) error {
			job, err := unmarshalMsgIntoEvent[model.Job](msg.Data())
			if err != nil {
				return err
			}
			return c.executor.ExecuteJob(ctx, job)
		})
}