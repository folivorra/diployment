package main

import (
	"context"
	flog "log"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/folivorra/diployment/internal/builder"
	"github.com/folivorra/diployment/internal/config"
	consumernats "github.com/folivorra/diployment/internal/consumer/nats"
	"github.com/folivorra/diployment/internal/minio"
	natsconn "github.com/folivorra/diployment/internal/nats"
	publishernats "github.com/folivorra/diployment/internal/publisher/nats"
	"github.com/folivorra/diployment/pkg/logger"

	"github.com/google/uuid"
	"github.com/moby/moby/client"
)

func main() {
	cfg := config.MustGetBuilder("config/.builder.env")

	log := logger.Setup(cfg.Env)
	slog.SetDefault(log)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	workerID := uuid.New().String()
	log.Info("starting builder", slog.String("worker_id", workerID))

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		flog.Fatalf("cannot create docker client: %v", err)
	}
	defer func() { _ = dockerClient.Close() }()

	minioClient, err := minio.NewMinioClient(cfg.MinIO)
	if err != nil {
		flog.Fatalf("cannot create minio client: %v", err)
	}
	if err := minio.InitBucket(ctx, minioClient, builder.BucketArtifacts); err != nil {
		flog.Fatalf("cannot init minio bucket: %v", err)
	}
	if err := minio.InitBucket(ctx, minioClient, builder.BucketLogs); err != nil {
		flog.Fatalf("cannot init minio logs bucket: %v", err)
	}

	conn, js, err := natsconn.NewConn(cfg.NATS.URL)
	if err != nil {
		flog.Fatalf("cannot connect to nats: %v", err)
	}
	defer conn.Close()
	if err := natsconn.SetupStreams(ctx, js); err != nil {
		flog.Fatalf("cannot setup streams: %v", err)
	}

	publisher := publishernats.NewNatsPublisher(js)
	b := builder.NewBuilder(dockerClient, minioClient, publisher, cfg.MasterKey, cfg.BuildTimeout)
	svc := builder.NewBuilderService(workerID, b, publisher)
	consumer := consumernats.NewNatsBuildConsumer(js, svc)

	if err := consumer.StartConsumeBuildsDispatch(ctx); err != nil {
		slog.Error("builds dispatch consumer stopped", slog.Any("error", err))
	}

	log.Info("shutting down builder", slog.String("worker_id", workerID))
}
