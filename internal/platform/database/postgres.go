package database

import (
	"context"
	"fmt"
	"log/slog"

	"educore/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB encapsulates the pgxpool connection pool and logger.
type DB struct {
	Pool   *pgxpool.Pool
	logger *slog.Logger
}

// New initializes, configures, and verifies a PostgreSQL connection pool using pgxpool.
// If the initial ping fails, the pool is closed and an error is returned.
func New(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database DSN configuration: %w", err)
	}

	// Apply connection pool settings
	poolConfig.MaxConns = int32(cfg.DBMaxConns)
	poolConfig.MinConns = int32(cfg.DBMinConns)
	poolConfig.MaxConnLifetime = cfg.DBMaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.DBMaxConnIdleTime

	// Connect with bounded timeout
	connectCtx, cancel := context.WithTimeout(ctx, cfg.DBConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection pool: %w", err)
	}

	// Verify database connection via Ping
	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	logger.Info("PostgreSQL connection pool established",
		slog.String("host", cfg.DBHost),
		slog.Int("port", cfg.DBPort),
		slog.String("database", cfg.DBName),
		slog.Int("max_conns", cfg.DBMaxConns),
		slog.Int("min_conns", cfg.DBMinConns),
		slog.Duration("max_conn_lifetime", cfg.DBMaxConnLifetime),
	)

	return &DB{
		Pool:   pool,
		logger: logger,
	}, nil
}

// Close gracefully closes all connections in the pool.
func (db *DB) Close() {
	if db != nil && db.Pool != nil {
		db.logger.Info("Closing PostgreSQL connection pool...")
		db.Pool.Close()
		db.logger.Info("PostgreSQL connection pool closed")
	}
}
