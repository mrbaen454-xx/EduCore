package integration_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"educore/internal/config"
	"educore/internal/platform/database"
	appErrors "educore/internal/shared/errors"
)

func getTestDBConfig() *config.Config {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := 5432
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "postgres"
	}
	dbName := os.Getenv("DB_TEST_NAME")
	if dbName == "" {
		dbName = "educore_test"
	}

	return &config.Config{
		AppEnv:              "testing",
		AppPort:             8080,
		AppTimezone:         "Asia/Jakarta",
		LogLevel:            "error",
		HTTPReadTimeout:     5 * time.Second,
		HTTPWriteTimeout:    5 * time.Second,
		HTTPIdleTimeout:     10 * time.Second,
		HTTPShutdownTimeout: 5 * time.Second,
		DBHost:              host,
		DBPort:              port,
		DBName:              dbName,
		DBUser:              user,
		DBPassword:          password,
		DBSSLMode:           "disable",
		DBMaxConns:          5,
		DBMinConns:          1,
		DBMaxConnLifetime:   10 * time.Minute,
		DBMaxConnIdleTime:   2 * time.Minute,
		DBConnectTimeout:    5 * time.Second,
		DBHealthTimeout:     2 * time.Second,
		JWTSecret:           "test-secret-key-that-is-at-least-32-chars-long!",
		JWTIssuer:           "educore-api",
		JWTAudience:         "educore-app",
		JWTAccessTokenTTL:   15 * time.Minute,
		JWTRefreshTokenTTL:  7 * 24 * time.Hour,
		Argon2Memory:        16 * 1024,
		Argon2Iterations:    1,
		Argon2Parallelism:   1,
		Argon2SaltLength:    16,
		Argon2KeyLength:     32,
	}
}

func TestDatabase_ConnectAndPing(t *testing.T) {
	cfg := getTestDBConfig()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.New(ctx, cfg, logger)
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	if db.Pool == nil {
		t.Fatal("expected non-nil pgxpool.Pool")
	}

	// Test health check helper
	if err := database.CheckHealth(ctx, db.Pool, cfg.DBHealthTimeout); err != nil {
		t.Errorf("expected health check to pass, got: %v", err)
	}
}

func TestDatabase_HealthCheck_NilPool(t *testing.T) {
	ctx := context.Background()
	err := database.CheckHealth(ctx, nil, 2*time.Second)
	if err == nil {
		t.Fatal("expected error for nil pool, got nil")
	}
}

func TestDatabase_ConnectionFailure_InvalidPort(t *testing.T) {
	cfg := getTestDBConfig()
	cfg.DBPort = 59999 // Non-existent port
	cfg.DBConnectTimeout = 1 * time.Second
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := database.New(ctx, cfg, logger)
	if err == nil {
		t.Fatal("expected connection error for unreachable port, got nil")
	}
}

func TestDatabase_ParseDBError_Generic(t *testing.T) {
	appErr := database.ParseDBError(nil)
	if appErr != nil {
		t.Errorf("expected nil for nil error, got %v", appErr)
	}

	stdErr := appErrors.ErrNotFound("Item not found")
	parsed := database.ParseDBError(stdErr)
	if parsed.HTTPStatus != http.StatusNotFound {
		t.Errorf("expected HTTP 404, got %d", parsed.HTTPStatus)
	}
}
