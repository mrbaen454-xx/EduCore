package database_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"educore/internal/platform/database"
	appErrors "educore/internal/shared/errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestParseDBError(t *testing.T) {
	// Nil error
	if database.ParseDBError(nil) != nil {
		t.Errorf("expected nil for nil input")
	}

	// Standard Go error
	stdErr := errors.New("generic error")
	appErr := database.ParseDBError(stdErr)
	if appErr.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("expected 500 for generic error, got %d", appErr.HTTPStatus)
	}

	// Unique Violation (23505)
	uniqueErr := &pgconn.PgError{Code: pgerrcode.UniqueViolation, Message: "duplicate key value"}
	uniqueAppErr := database.ParseDBError(uniqueErr)
	if uniqueAppErr.HTTPStatus != http.StatusConflict {
		t.Errorf("expected 409 Conflict, got %d", uniqueAppErr.HTTPStatus)
	}
	if uniqueAppErr.Code != appErrors.CodeConflict {
		t.Errorf("expected code CONFLICT, got %s", uniqueAppErr.Code)
	}

	// Foreign Key Violation (23503)
	fkErr := &pgconn.PgError{Code: pgerrcode.ForeignKeyViolation, Message: "violates foreign key constraint"}
	fkAppErr := database.ParseDBError(fkErr)
	if fkAppErr.HTTPStatus != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for FK violation, got %d", fkAppErr.HTTPStatus)
	}

	// Check Violation (23514)
	checkErr := &pgconn.PgError{Code: pgerrcode.CheckViolation, Message: "violates check constraint"}
	checkAppErr := database.ParseDBError(checkErr)
	if checkAppErr.HTTPStatus != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for check violation, got %d", checkAppErr.HTTPStatus)
	}

	// Not Null Violation (23502)
	nnErr := &pgconn.PgError{Code: pgerrcode.NotNullViolation, Message: "null value in column violates not-null"}
	nnAppErr := database.ParseDBError(nnErr)
	if nnAppErr.HTTPStatus != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for not-null violation, got %d", nnAppErr.HTTPStatus)
	}

	// Admin Shutdown / Connection Error (57P01)
	shutdownErr := &pgconn.PgError{Code: pgerrcode.AdminShutdown, Message: "terminating connection due to administrator command"}
	shutdownAppErr := database.ParseDBError(shutdownErr)
	if shutdownAppErr.HTTPStatus != http.StatusServiceUnavailable {
		t.Errorf("expected 503 Service Unavailable, got %d", shutdownAppErr.HTTPStatus)
	}
}

func TestCheckHealth_NilPool(t *testing.T) {
	ctx := context.Background()
	err := database.CheckHealth(ctx, nil, 1*time.Second)
	if err == nil {
		t.Fatalf("expected error for nil pool, got nil")
	}
}

func TestRunInTx_NilPool(t *testing.T) {
	ctx := context.Background()
	err := database.RunInTx(ctx, nil, func(tx pgx.Tx) error {
		return nil
	})
	if err == nil {
		t.Fatalf("expected error for nil pool in RunInTx, got nil")
	}
}
