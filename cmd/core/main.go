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
	"github.com/folivorra/diployment/internal/core/handler"
	"github.com/folivorra/diployment/internal/core/provider"
	"github.com/folivorra/diployment/internal/core/repository/postgres"
	"github.com/folivorra/diployment/internal/core/service"
	"github.com/folivorra/diployment/internal/pgpool"
	"github.com/folivorra/diployment/pkg/logger"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	slogecho "github.com/samber/slog-echo"
)

const shutdownTimeout = 10 * time.Second

func main() {
	cfg := config.MustGet()

	log := logger.Setup(cfg.Env)
	slog.SetDefault(log)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := pgpool.NewPool(ctx, cfg.Postgres.DSN)
	if err != nil {
		flog.Fatalf("cannot create pgx pool: %v", err)
	}

	userRepo := postgres.NewUserPostgresRepo(pool)
	githubProvider := provider.NewGitHubProvider(cfg.GitHub)
	authService := service.NewAuthService(githubProvider, userRepo, cfg.Auth, cfg.MasterKey)
	authHandler := handler.NewAuthHandler(authService)

	e := echo.New()

	e.Use(slogecho.New(log))                               // чтобы каждый HTTP-запрос логировался в slog
	e.Use(middleware.Recover())                            // чтобы сервер не падал на панике, а писал ошибку в логи
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{ // для корректной работы фронт-бэк
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowCredentials: true, // важно для работы с куками!
	}))

	e.GET("/auth/login", authHandler.Login)              // redirect to GitHub
	e.GET("/auth/github/callback", authHandler.Callback) // callback to get user info from GitHub

	log.Info("starting server", slog.String("address", cfg.HTTP.Address()), slog.Any("env_mode", cfg.Env))

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
