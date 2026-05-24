package main

import (
	"context"
	flog "log"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/folivorra/diployment/internal/config"
	consumernats "github.com/folivorra/diployment/internal/consumer/nats"
	"github.com/folivorra/diployment/internal/deployer"
	"github.com/folivorra/diployment/internal/minio"
	natsconn "github.com/folivorra/diployment/internal/nats"
	publishernats "github.com/folivorra/diployment/internal/publisher/nats"
	"github.com/folivorra/diployment/pkg/logger"

	"github.com/google/uuid"
)

func main() {
	cfg := config.MustGetDeployer("config/.deployer.env")

	log := logger.Setup(cfg.Env)
	slog.SetDefault(log)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	workerID := uuid.New().String()
	log.Info("starting deployer", slog.String("worker_id", workerID))

	minioClient, err := minio.NewMinioClient(cfg.MinIO)
	if err != nil {
		flog.Fatalf("cannot create minio client: %v", err)
	}
	if err := minio.InitBucket(ctx, minioClient, deployer.BucketArtifacts); err != nil {
		flog.Fatalf("cannot init minio artifacts bucket: %v", err)
	}
	if err := minio.InitBucket(ctx, minioClient, deployer.BucketLogs); err != nil {
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
	d := deployer.NewDeployer(minioClient, publisher, cfg.MasterKey, cfg.DeployTimeout, cfg.SSHDialTimeout)
	svc := deployer.NewDeployerService(workerID, d, publisher)
	consumer := consumernats.NewNatsDeployerConsumer(js, svc)

	if err := consumer.StartConsumeDeployDispatch(ctx); err != nil {
		slog.Error("deploy dispatch consumer stopped", slog.Any("error", err))
	}

	log.Info("shutting down deployer", slog.String("worker_id", workerID))
}
