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

	// 2. Check Migration Version (now version 6 with study_plans and enrollments tables)
	version, dirty, err := database.MigrationVersion(migrationsDir, cfg.DSN())
	if err != nil {
		t.Fatalf("failed to get migration version: %v", err)
	}
	if dirty {
		t.Errorf("migration state is dirty")
	}
	if version != 6 {
		t.Errorf("expected migration version 6, got %d", version)
	}

	// 3. Verify all tables exist in PostgreSQL information_schema
	tables := []string{
		"roles", "users", "user_roles", "refresh_tokens",
		"faculties", "study_programs", "lecturers", "students",
		"courses", "curricula", "curriculum_courses", "course_prerequisites",
		"academic_periods", "course_offerings", "classes", "class_lecturers", "class_schedules",
		"study_plans", "enrollments",
	}
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

	// 5. Test MigrateDown (Rollback step 1: drop study_plans & enrollments tables)
	if err := database.MigrateDown(migrationsDir, cfg.DSN(), 1); err != nil {
		t.Fatalf("failed to run MigrateDown step 1: %v", err)
	}

	for _, table := range []string{"enrollments", "study_plans"} {
		var exists bool
		_ = db.Pool.QueryRow(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema='public' AND table_name=$1);", table).Scan(&exists)
		if exists {
			t.Errorf("expected table %s to be dropped after rollback step 1", table)
		}
	}

	// 6. Rollback step 2: drop classes & schedules tables
	if err := database.MigrateDown(migrationsDir, cfg.DSN(), 1); err != nil {
		t.Fatalf("failed to run MigrateDown step 2: %v", err)
	}

	for _, table := range []string{"class_schedules", "class_lecturers", "classes", "course_offerings", "academic_periods"} {
		var exists bool
		_ = db.Pool.QueryRow(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema='public' AND table_name=$1);", table).Scan(&exists)
		if exists {
			t.Errorf("expected table %s to be dropped after rollback step 2", table)
		}
	}

	// 7. Rollback step 3: drop courses & curricula tables
	if err := database.MigrateDown(migrationsDir, cfg.DSN(), 1); err != nil {
		t.Fatalf("failed to run MigrateDown step 3: %v", err)
	}

	for _, table := range []string{"course_prerequisites", "curriculum_courses", "curricula", "courses"} {
		var exists bool
		_ = db.Pool.QueryRow(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema='public' AND table_name=$1);", table).Scan(&exists)
		if exists {
			t.Errorf("expected table %s to be dropped after rollback step 3", table)
		}
	}

	// 8. Rollback step 4: drop academic master tables
	if err := database.MigrateDown(migrationsDir, cfg.DSN(), 1); err != nil {
		t.Fatalf("failed to run MigrateDown step 4: %v", err)
	}

	for _, table := range []string{"students", "lecturers", "study_programs", "faculties"} {
		var exists bool
		_ = db.Pool.QueryRow(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema='public' AND table_name=$1);", table).Scan(&exists)
		if exists {
			t.Errorf("expected table %s to be dropped after rollback step 4", table)
		}
	}

	// 9. Rollback step 5: drop refresh_tokens
	if err := database.MigrateDown(migrationsDir, cfg.DSN(), 1); err != nil {
		t.Fatalf("failed to run MigrateDown step 5: %v", err)
	}

	var rtExists bool
	_ = db.Pool.QueryRow(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema='public' AND table_name='refresh_tokens');").Scan(&rtExists)
	if rtExists {
		t.Errorf("expected refresh_tokens to be dropped after rollback step 5")
	}

	// 10. Rollback step 6: drop initial schema
	if err := database.MigrateDown(migrationsDir, cfg.DSN(), 1); err != nil {
		t.Fatalf("failed to run MigrateDown step 6: %v", err)
	}

	for _, table := range []string{"roles", "users", "user_roles"} {
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

	// 11. Re-apply MigrateUp to leave schema active for subsequent tests
	if err := database.MigrateUp(migrationsDir, cfg.DSN()); err != nil {
		t.Fatalf("failed to re-apply MigrateUp: %v", err)
	}
}
