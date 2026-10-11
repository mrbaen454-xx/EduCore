package integration_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"educore/internal/platform/database"
	appErrors "educore/internal/shared/errors"

	"github.com/jackc/pgx/v5"
)

func TestTransaction_CommitAndRollback(t *testing.T) {
	cfg := getTestDBConfig()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.New(ctx, cfg, logger)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	defer db.Close()

	// Ensure schema is up
	_ = database.MigrateUp("../../migrations", cfg.DSN())

	// 1. Successful transaction with commit
	var userID string
	err = database.RunInTx(ctx, db.Pool, func(tx pgx.Tx) error {
		query := `INSERT INTO users (email, password_hash, name) 
			VALUES ($1, $2, $3) 
			RETURNING id;`
		if err := tx.QueryRow(ctx, query, "tx_success@educore.ac.id", "hashed_pw_123", "User Transaksi").Scan(&userID); err != nil {
			return err
		}

		assignRole := `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2);`
		_, err := tx.Exec(ctx, assignRole, userID, "dosen")
		return err
	})

	if err != nil {
		t.Fatalf("expected transaction to commit successfully, got: %v", err)
	}

	// Verify user persisted
	var count int
	_ = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = $1;", "tx_success@educore.ac.id").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 user to be persisted after commit, got %d", count)
	}

	// 2. Failed transaction with rollback
	intentionalErr := errors.New("simulated business rule validation error")
	err = database.RunInTx(ctx, db.Pool, func(tx pgx.Tx) error {
		query := `INSERT INTO users (email, password_hash, name) 
			VALUES ($1, $2, $3);`
		if _, err := tx.Exec(ctx, query, "tx_rollback@educore.ac.id", "hashed_pw_123", "User Rollback"); err != nil {
			return err
		}
		// Return error to trigger rollback
		return intentionalErr
	})

	if !errors.Is(err, intentionalErr) {
		t.Fatalf("expected intentionalErr to be returned, got: %v", err)
	}

	// Verify user was NOT persisted after rollback
	var rollbackCount int
	_ = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = $1;", "tx_rollback@educore.ac.id").Scan(&rollbackCount)
	if rollbackCount != 0 {
		t.Errorf("expected 0 users after rollback, got %d", rollbackCount)
	}
}

func TestDatabase_Constraints(t *testing.T) {
	cfg := getTestDBConfig()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.New(ctx, cfg, logger)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	defer db.Close()

	// 1. Unique Email Constraint
	uniqueEmail := "unique_test@educore.ac.id"
	_, _ = db.Pool.Exec(ctx, "INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3);", uniqueEmail, "hash", "User 1")

	// Insert duplicate email
	_, dupErr := db.Pool.Exec(ctx, "INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3);", uniqueEmail, "hash", "User 2")
	if dupErr == nil {
		t.Fatal("expected unique constraint violation error, got nil")
	}

	parsedDup := database.ParseDBError(dupErr)
	if parsedDup.HTTPStatus != http.StatusConflict {
		t.Errorf("expected 409 Conflict for unique violation, got %d", parsedDup.HTTPStatus)
	}
	if parsedDup.Code != appErrors.CodeConflict {
		t.Errorf("expected code CONFLICT, got %s", parsedDup.Code)
	}

	// 2. Foreign Key Constraint on user_roles
	var validUserID string
	_ = db.Pool.QueryRow(ctx, "SELECT id FROM users WHERE email = $1;", uniqueEmail).Scan(&validUserID)

	// Attempt to assign non-existent role
	_, fkErr := db.Pool.Exec(ctx, "INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2);", validUserID, "non_existent_role_xyz")
	if fkErr == nil {
		t.Fatal("expected foreign key violation error, got nil")
	}

	parsedFK := database.ParseDBError(fkErr)
	if parsedFK.HTTPStatus != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for FK violation, got %d", parsedFK.HTTPStatus)
	}

	// 3. Check Constraint on users email (empty email)
	_, checkErr := db.Pool.Exec(ctx, "INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3);", "   ", "hash", "User Check")
	if checkErr == nil {
		t.Fatal("expected check constraint violation error for empty email, got nil")
	}

	parsedCheck := database.ParseDBError(checkErr)
	if parsedCheck.HTTPStatus != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for check violation, got %d", parsedCheck.HTTPStatus)
	}

	// 4. Not-Null Constraint on users password_hash
	_, nnErr := db.Pool.Exec(ctx, "INSERT INTO users (email, password_hash, name) VALUES ($1, NULL, $2);", "null_pw@educore.ac.id", "User NN")
	if nnErr == nil {
		t.Fatal("expected not-null constraint violation error, got nil")
	}

	parsedNN := database.ParseDBError(nnErr)
	if parsedNN.HTTPStatus != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for not-null violation, got %d", parsedNN.HTTPStatus)
	}
}
