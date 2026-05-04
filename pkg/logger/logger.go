package logger

import (
	"log/slog"
	"os"

	"github.com/folivorra/diployment/internal/config"

	"github.com/lmittmann/tint"
)

// Setup устанавливает нужный, в зависимости от окружения, логгер (текстовый или JSON)
func Setup(env config.Env) *slog.Logger {
	var log *slog.Logger

	switch env {
	case config.EnvLocal:
		log = slog.New(tint.NewHandler(os.Stdout, &tint.Options{Level: slog.LevelDebug}))
	case config.EnvDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	return log
}
