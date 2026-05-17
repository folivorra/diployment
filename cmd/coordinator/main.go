package main

import (
	"context"
	flog "log"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/folivorra/diployment/internal/config"
	consumernats "github.com/folivorra/diployment/internal/consumer/nats"
	"github.com/folivorra/diployment/internal/coordinator"
	natsconn "github.com/folivorra/diployment/internal/nats"
	"github.com/folivorra/diployment/internal/pgpool"
	publishernats "github.com/folivorra/diployment/internal/publisher/nats"
	"github.com/folivorra/diployment/internal/repository/postgres"
	"github.com/folivorra/diployment/pkg/logger"
)

func main() {
	cfg := config.MustGetCoordinator("config/.coordinator.env")

	log := logger.Setup(cfg.Env)
	slog.SetDefault(log)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := pgpool.NewPool(ctx, cfg.Postgres.DSN)
	if err != nil {
		flog.Fatalf("cannot create pgx pool: %v", err)
	}

	conn, js, err := natsconn.NewConn(cfg.NATS.URL)
	if err != nil {
		flog.Fatalf("cannot connect to nats: %v", err)
	}
	defer conn.Close()

	jobRepo := postgres.NewJobPostgresRepo(pool)
	publisher := publishernats.NewNatsPublisher(js)
	svc := coordinator.NewCoordService(jobRepo, publisher)
	consumer := consumernats.NewNatsCoordinatorConsumer(js, svc)

	log.Info("starting coordinator")

	go func() {
		if err := consumer.StartConsumeBuilds(ctx); err != nil {
			slog.Error("builds consumer stopped", slog.Any("error", err))
		}
	}()

	go func() {
		if err := consumer.StartConsumeJobsStatus(ctx); err != nil {
			slog.Error("jobs consumer stopped", slog.Any("error", err))
		}
	}()

	<-ctx.Done()
	log.Info("shutting down coordinator")
}
