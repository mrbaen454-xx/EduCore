package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CheckHealth verifies that the PostgreSQL connection pool is alive and accepting queries.
// It uses a bounded timeout to prevent blocking HTTP readiness probes indefinitely.
func CheckHealth(ctx context.Context, pool *pgxpool.Pool, timeout time.Duration) error {
	if pool == nil {
		return fmt.Errorf("database connection pool is uninitialized (nil)")
	}

	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		return fmt.Errorf("database ping check failed: %w", err)
	}

	return nil
}
