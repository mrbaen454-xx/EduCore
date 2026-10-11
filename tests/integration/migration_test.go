package integration_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"educore/internal/platform/database"
)

func TestMigration_Lifecycle(t *testing.T) {
	cfg := getTestDBConfig()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.New(ctx, cfg, logger)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	defer db.Close()

	migrationsDir := "../../migrations"

	// 1. Run MigrateUp
	if err := database.MigrateUp(migrationsDir, cfg.DSN()); err != nil {
		t.Fatalf("failed to run MigrateUp: %v", err)
	}

	// 2. Check Migration Version
	version, dirty, err := database.MigrationVersion(migrationsDir, cfg.DSN())
	if err != nil {
		t.Fatalf("failed to get migration version: %v", err)
	}
	if dirty {
		t.Errorf("migration state is dirty")
	}
	if version != 1 {
		t.Errorf("expected migration version 1, got %d", version)
	}

	// 3. Verify tables exist in PostgreSQL information_schema
	tables := []string{"roles", "users", "user_roles"}
	for _, table := range tables {
		var exists bool
		query := `SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = $1
		);`
		if err := db.Pool.QueryRow(ctx, query, table).Scan(&exists); err != nil {
			t.Fatalf("failed to query table existence for %s: %v", table, err)
		}
		if !exists {
			t.Errorf("expected table %s to exist after migration", table)
		}
	}

	// 4. Verify baseline seed roles
	var roleCount int
	if err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM roles;").Scan(&roleCount); err != nil {
		t.Fatalf("failed to count roles: %v", err)
	}
	if roleCount != 4 {
		t.Errorf("expected 4 baseline roles (admin, dosen, mahasiswa, akademik), got %d", roleCount)
	}

	// 5. Test MigrateDown (Rollback)
	if err := database.MigrateDown(migrationsDir, cfg.DSN(), 1); err != nil {
		t.Fatalf("failed to run MigrateDown: %v", err)
	}

	// 6. Verify tables dropped after down migration
	for _, table := range tables {
		var exists bool
		query := `SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = $1
		);`
		if err := db.Pool.QueryRow(ctx, query, table).Scan(&exists); err != nil {
			t.Fatalf("failed to query table existence after rollback for %s: %v", table, err)
		}
		if exists {
			t.Errorf("expected table %s to be dropped after rollback", table)
		}
	}

	// 7. Re-apply MigrateUp to leave schema active for subsequent tests
	if err := database.MigrateUp(migrationsDir, cfg.DSN()); err != nil {
		t.Fatalf("failed to re-apply MigrateUp: %v", err)
	}
}
