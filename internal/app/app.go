package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"educore/internal/config"
	"educore/internal/platform/database"
	"educore/internal/platform/httpserver"
	"educore/internal/platform/logger"
)

// Run initializes and starts the application, managing its lifecycle.
func Run() error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	log := logger.New(cfg.LogLevel, cfg.AppEnv)
	log.Info("Starting EduCore application", slog.String("env", cfg.AppEnv))

	// Initialize database connection pool
	db, err := database.New(context.Background(), cfg, log)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Initialize HTTP server with database dependency
	server := httpserver.New(cfg, log, db.Pool)

	// Start server
	serverErrChan := server.Start()

	// Wait for interruption signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrChan:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		log.Info("Received shutdown signal", slog.String("signal", sig.String()))
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPShutdownTimeout)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Info("Application stopped gracefully")
	return nil
}
