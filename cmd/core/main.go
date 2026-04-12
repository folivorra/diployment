package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/folivorra/diployment/internal/config"
	"github.com/folivorra/diployment/pkg/logger"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	slogecho "github.com/samber/slog-echo"
)

func main() {
	cfg := config.MustGet()

	log := logger.Setup(cfg.Env)
	slog.SetDefault(log)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	e := echo.New()

	e.Use(slogecho.New(log))    // чтобы каждый HTTP-запрос логировался в slog
	e.Use(middleware.Recover()) // чтобы сервер не падал на панике, а писал ошибку в логи

	log.Info("starting server", slog.String("address", cfg.HTTP.Address()))

	eCfg := echo.StartConfig{Address: cfg.HTTP.Address()}
	go func() {
		if err := eCfg.Start(ctx, e); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Error("server failed to start: %v", slog.Any("error", err))
		}
	}()

	<-ctx.Done()
	log.Info("shutting down server gracefully")
}
