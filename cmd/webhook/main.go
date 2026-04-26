package main

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/folivorra/diployment/internal/config"
	"github.com/folivorra/diployment/pkg/logger"
)

func main() {
	cfg := config.MustGetWebhook("config/.webhook.env")

	log := logger.Setup(cfg.Env)
	slog.SetDefault(log)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	<-ctx.Done()
	log.Info("shutting down server gracefully")
}
