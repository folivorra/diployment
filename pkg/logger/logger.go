package logger

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

const (
	envLocal = "local"
	envDev   = "dev"
)

// Setup устанавливает нужный, в зависимости от окружения, логгер (текстовый или JSON)
func Setup(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(tint.NewHandler(os.Stdout, &tint.Options{Level: slog.LevelDebug}))
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	return log
}
