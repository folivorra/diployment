package main

import (
	"context"
	flog "log"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/folivorra/diployment/internal/config"
	consumernats "github.com/folivorra/diployment/internal/consumer/nats"
	natsconn "github.com/folivorra/diployment/internal/nats"
	publishernats "github.com/folivorra/diployment/internal/publisher/nats"
	"github.com/folivorra/diployment/internal/worker"
	"github.com/folivorra/diployment/pkg/logger"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/moby/moby/client"
)

func main() {
	cfg := config.MustGetWorker("config/.worker.env")

	log := logger.Setup(cfg.Env)
	slog.SetDefault(log)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	workerID := uuid.New().String()
	log.Info("starting worker", slog.String("worker_id", workerID))

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		flog.Fatalf("cannot create docker client: %v", err)
	}
	defer func() { _ = dockerClient.Close() }()

	minioClient, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		flog.Fatalf("cannot create minio client: %v", err)
	}

	conn, js, err := natsconn.NewConn(cfg.NATS.URL)
	if err != nil {
		flog.Fatalf("cannot connect to nats: %v", err)
	}
	defer conn.Close()

	publisher := publishernats.NewNatsPublisher(js)
	builder := worker.NewBuilder(dockerClient, minioClient)
	svc := worker.NewWorkerService(workerID, builder, publisher)
	consumer := consumernats.NewNatsQueueWorkerConsumer(js, svc)

	if err := consumer.StartConsumeJobsDispatch(ctx); err != nil {
		slog.Error("worker consumer stopped", slog.Any("error", err))
	}

	log.Info("shutting down worker", slog.String("worker_id", workerID))
}
