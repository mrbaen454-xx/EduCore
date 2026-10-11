package database

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// formatMigrateDSN ensures the DSN uses the pgx5 driver scheme recognized by golang-migrate
func formatMigrateDSN(dsn string) string {
	if strings.HasPrefix(dsn, "postgres://") {
		return strings.Replace(dsn, "postgres://", "pgx5://", 1)
	}
	if strings.HasPrefix(dsn, "postgresql://") {
		return strings.Replace(dsn, "postgresql://", "pgx5://", 1)
	}
	return dsn
}

// MigrateUp applies all pending database migrations from migrationsDir.
func MigrateUp(migrationsDir string, dsn string) error {
	migrateDSN := formatMigrateDSN(dsn)
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsDir),
		migrateDSN,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migration instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations up: %w", err)
	}

	return nil
}

// MigrateDown rolls back the last n database migrations.
func MigrateDown(migrationsDir string, dsn string, steps int) error {
	migrateDSN := formatMigrateDSN(dsn)
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsDir),
		migrateDSN,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migration instance: %w", err)
	}
	defer m.Close()

	if err := m.Steps(-steps); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to rollback migrations down: %w", err)
	}

	return nil
}

// MigrationVersion returns the current database migration version and whether it is dirty.
func MigrationVersion(migrationsDir string, dsn string) (uint, bool, error) {
	migrateDSN := formatMigrateDSN(dsn)
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsDir),
		migrateDSN,
	)
	if err != nil {
		return 0, false, fmt.Errorf("failed to initialize migration instance: %w", err)
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, dirty, nil
}
