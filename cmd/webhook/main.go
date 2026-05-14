package main

import (
	"context"
	"errors"
	flog "log"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/folivorra/diployment/internal/config"
	natsconn "github.com/folivorra/diployment/internal/nats"
	"github.com/folivorra/diployment/internal/pgpool"
	"github.com/folivorra/diployment/internal/publisher/nats"
	"github.com/folivorra/diployment/internal/repository/postgres"
	"github.com/folivorra/diployment/internal/webhook"
	"github.com/folivorra/diployment/pkg/logger"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	slogecho "github.com/samber/slog-echo"
)

const shutdownTimeout = 10 * time.Second

func main() {
	cfg := config.MustGetWebhook("config/.webhook.env")

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
		flog.Fatalf("cannot connect nats: %v", err)
	}
	defer conn.Close()
	if err := natsconn.SetupStreams(ctx, js); err != nil {
		flog.Fatalf("cannot setup streams: %v", err)
	}

	natsPub := nats.NewNatsPublisher(js)
	projectRepo := postgres.NewProjectPostgresRepo(pool)
	userRepo := postgres.NewUserPostgresRepo(pool)
	handler := webhook.NewWebhookHandler(projectRepo, natsPub, userRepo, cfg.MasterKey)

	e := echo.New()

	e.Use(slogecho.New(log))    // чтобы каждый HTTP-запрос логировался в slog
	e.Use(middleware.Recover()) // чтобы сервер не падал на панике, а писал ошибку в логи

	e.POST("/webhook", handler.Webhook)

	go func() {
		if err := e.Start(cfg.HTTP.Address()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Errorf("server failed to start: %v", slog.Any("error", err))
		}
	}()

	<-ctx.Done()
	log.Info("shutting down server gracefully")

	shuttingDownCtx, shuttingDownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shuttingDownCancel()

	if err := e.Shutdown(shuttingDownCtx); err != nil {
		e.Logger.Fatal(err)
	}
}
