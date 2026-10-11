package logger

import (
	"log/slog"
	"os"
)

// New creates and configures a new structured logger based on environment.
func New(levelStr string, env string) *slog.Logger {
	var level slog.Level
	switch levelStr {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if env == "development" {
		// Use TextHandler for better readability locally
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		// Use JSONHandler for production/structured logging
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
