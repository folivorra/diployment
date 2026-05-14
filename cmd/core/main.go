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
	authmddlwr "github.com/folivorra/diployment/internal/core/handler/middleware"
	"github.com/folivorra/diployment/internal/core/provider"
	"github.com/folivorra/diployment/internal/core/service"
	"github.com/folivorra/diployment/internal/pgpool"
	"github.com/folivorra/diployment/internal/repository/postgres"
	"github.com/folivorra/diployment/pkg/logger"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	slogecho "github.com/samber/slog-echo"
)

const shutdownTimeout = 10 * time.Second

func main() {
	cfg := config.MustGetCore("config/.core.env")

	log := logger.Setup(cfg.Env)
	slog.SetDefault(log)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := pgpool.NewPool(ctx, cfg.Postgres.DSN)
	if err != nil {
		flog.Fatalf("cannot create pgx pool: %v", err)
	}

	userRepo := postgres.NewUserPostgresRepo(pool)
	projectRepo := postgres.NewProjectPostgresRepo(pool)
	githubProvider := provider.NewGitHubProvider(cfg.GitHub, cfg.Webhook.URL)
	repoService := service.NewRepoService(githubProvider, userRepo, cfg.MasterKey)
	authService := service.NewAuthService(githubProvider, userRepo, cfg.Auth, cfg.MasterKey)
	projectService := service.NewProjectService(projectRepo, githubProvider, userRepo, cfg.MasterKey)
	repoHandler := handler.NewRepoHandler(repoService)
	authHandler := handler.NewAuthHandler(authService)
	projectHandler := handler.NewProjectHandler(projectService)

	e := echo.New()

	e.Use(slogecho.New(log))                               // чтобы каждый HTTP-запрос логировался в slog
	e.Use(middleware.Recover())                            // чтобы сервер не падал на панике, а писал ошибку в логи
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{ // для корректной работы фронт-бэк
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowCredentials: true, // важно для работы с куками!
	}))

	auth := e.Group("/auth")
	{
		auth.GET("/login", authHandler.Login)       // редиректит на провайдера
		auth.GET("/callback", authHandler.Callback) // callback для получения инфо о юзере GitHub
	}

	api := e.Group("/api")
	{
		api.Use(authmddlwr.AuthMiddleware(cfg.Auth.JWTSecret)) // проверяет jwt токен и кладет user_id в контекст

		api.GET("/repos", repoHandler.ListRepos)           // получаем список репозиториев пользователя
		api.POST("/project/import", projectHandler.Import) // импортируем репо в проект
	}

	go func() {
		if err := e.Start(cfg.HTTP.Address()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Errorf("server failed to start: %v", slog.Any("error", err))
		}
	}()

	<-ctx.Done()
	log.Info("shutting down server gracefully")

	shuttingDownCtx, shuttingDownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shuttingDownCancel()

	if err = e.Shutdown(shuttingDownCtx); err != nil {
		e.Logger.Fatal(err)
	}
}
