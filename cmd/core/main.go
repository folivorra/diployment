package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/folivorra/diployment/internal/config"
	"github.com/folivorra/diployment/pkg/logger"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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

	log.Info("starting server", slog.String("address", cfg.HTTP.Address()), slog.Any("env_mode", cfg.Env))

	go func() {
		if err := e.Start(cfg.HTTP.Address()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Error("server failed to start: %v", slog.Any("error", err))
		}
	}()

	<-ctx.Done()
	log.Info("shutting down server gracefully")

	shuttingDownCtx, shuttingDownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shuttingDownCancel()

	// 4. Корректно останавливаем сервер
	if err := e.Shutdown(shuttingDownCtx); err != nil {
		e.Logger.Fatal(err)
	}
}
