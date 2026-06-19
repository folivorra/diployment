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
	minioclient "github.com/folivorra/diployment/internal/minio"
	natsconn "github.com/folivorra/diployment/internal/nats"
	"github.com/folivorra/diployment/internal/pgpool"
	"github.com/folivorra/diployment/internal/repository/postgres"
	miniorepo "github.com/folivorra/diployment/internal/repository/minio"
	subscribernats "github.com/folivorra/diployment/internal/subscriber/nats"
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

	conn, js, err := natsconn.NewConn(cfg.NATS.URL)
	if err != nil {
		flog.Fatalf("cannot connect to nats: %v", err)
	}
	defer conn.Close()
	if err := natsconn.SetupStreams(ctx, js); err != nil {
		flog.Fatalf("cannot setup streams: %v", err)
	}

	minioClient, err := minioclient.NewMinioClient(cfg.MinIO)
	if err != nil {
		flog.Fatalf("cannot create minio client: %v", err)
	}

	userRepo := postgres.NewUserPostgresRepo(pool)
	projectRepo := postgres.NewProjectPostgresRepo(pool)
	jobRepo := postgres.NewJobPostgresRepo(pool)
	logRepo := miniorepo.NewLogMinioRepo(minioClient, miniorepo.BucketLogs)
	githubProvider := provider.NewGitHubProvider(cfg.GitHub, cfg.Webhook.URL)
	repoService := service.NewRepoService(githubProvider, userRepo, cfg.MasterKey)
	authService := service.NewAuthService(githubProvider, userRepo, cfg.Auth, cfg.MasterKey)
	userService := service.NewUserService(userRepo)
	projectService := service.NewProjectService(projectRepo, projectRepo, githubProvider, userRepo, cfg.MasterKey)
	jobSubscriber := subscribernats.NewSubscriberNats(js)
	jobService := service.NewJobService(jobRepo, logRepo)
	repoHandler := handler.NewRepoHandler(repoService)
	authHandler := handler.NewAuthHandler(authService, cfg.Frontend.URL)
	userHandler := handler.NewUserHandler(userService)
	projectHandler := handler.NewProjectHandler(projectService)
	jobHandler := handler.NewJobHandler(jobService, jobSubscriber, jobSubscriber)

	e := echo.New()

	e.Use(slogecho.New(log))    // чтобы каждый HTTP-запрос логировался в slog
	e.Use(middleware.Recover()) // чтобы сервер не падал на панике, а писал ошибку в логи
	e.Use(middleware.CORSWithConfig(
		middleware.CORSConfig{ // для корректной работы фронт-бэк
			AllowOrigins:     []string{"http://localhost:3000", cfg.Frontend.URL},
			AllowCredentials: true, // важно для работы с куками!
		},
	))

	e.GET("/health", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	auth := e.Group("/auth")
	{
		auth.GET("/login", authHandler.Login)       // редиректит на провайдера
		auth.GET("/callback", authHandler.Callback) // callback для получения инфо о юзере GitHub
	}

	api := e.Group("/api")
	api.Use(authmddlwr.AuthMiddleware(cfg.Auth.JWTSecret)) // проверяет jwt токен и кладет user_id в контекст
	{
		api.GET("/me", userHandler.Me)                          // информация о текущем пользователе
		api.GET("/repos", repoHandler.ListRepos)                // список репозиториев пользователя
		api.GET("/repos/branches", repoHandler.ListBranches)    // список веток репозитория на GitHub
		api.GET("/projects", projectHandler.List)               // список проектов пользователя
		api.POST("/projects/import", projectHandler.Import)     // импортировать репозиторий как проект
		api.GET("/projects/:id/jobs", jobHandler.ListByProject)    // история джоб проекта
		api.GET("/jobs/:id/events", jobHandler.Events)             // SSE стрим статусов джобы
		api.GET("/jobs/:id/logs/:phase", jobHandler.GetLog)        // персистентный лог из MinIO
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
