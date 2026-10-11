package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"educore/internal/config"
	"educore/internal/modules/auth"
	"educore/internal/platform/database"
	"educore/internal/platform/httpserver"
	"educore/internal/platform/logger"
	"educore/internal/shared/response"

	"github.com/google/uuid"
)

func setupAuthIntegrationApp(t *testing.T) (*httpserver.Server, *auth.Repository, *config.Config) {
	cfg := getTestDBConfig()
	log := logger.New("error", "testing")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.New(ctx, cfg, log)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}

	// Apply migrations
	_ = database.MigrateUp("../../migrations", cfg.DSN())

	hasher := auth.NewArgon2Hasher(16*1024, 1, 1, 16, 32)
	repo := auth.NewRepository(db.Pool)

	// Clean test data before tests
	_, _ = db.Pool.Exec(context.Background(), "DELETE FROM refresh_tokens;")
	_, _ = db.Pool.Exec(context.Background(), "DELETE FROM user_roles;")
	_, _ = db.Pool.Exec(context.Background(), "DELETE FROM users;")

	// Provision test users
	adminPassHash, _ := hasher.Hash("AdminSecret123!")
	adminUser := &auth.User{
		ID:           uuid.New(),
		Email:        "admin@example.com",
		PasswordHash: adminPassHash,
		Name:         "Super Administrator",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_ = repo.CreateUser(context.Background(), adminUser, []string{"admin"})

	studentPassHash, _ := hasher.Hash("StudentSecret123!")
	studentUser := &auth.User{
		ID:           uuid.New(),
		Email:        "student@example.com",
		PasswordHash: studentPassHash,
		Name:         "Budi Santoso",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_ = repo.CreateUser(context.Background(), studentUser, []string{"mahasiswa"})

	inactivePassHash, _ := hasher.Hash("InactiveSecret123!")
	inactiveUser := &auth.User{
		ID:           uuid.New(),
		Email:        "inactive@example.com",
		PasswordHash: inactivePassHash,
		Name:         "Inactive User",
		IsActive:     false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_ = repo.CreateUser(context.Background(), inactiveUser, []string{"mahasiswa"})

	server := httpserver.New(cfg, log, db.Pool)
	return server, repo, cfg
}

func TestAuth_LoginSuccess(t *testing.T) {
	server, _, _ := setupAuthIntegrationApp(t)

	loginPayload := auth.LoginRequest{
		Identifier: "admin@example.com",
		Password:   "AdminSecret123!",
	}
	body, _ := json.Marshal(loginPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res struct {
		Success bool               `json:"success"`
		Message string             `json:"message"`
		Data    auth.LoginResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

	if !res.Success {
		t.Errorf("expected success true, got false")
	}
	if len(res.Data.AccessToken) == 0 {
		t.Error("expected non-empty access token")
	}
	if res.Data.TokenType != "Bearer" {
		t.Errorf("expected token type Bearer, got %s", res.Data.TokenType)
	}
	if len(res.Data.RefreshToken) == 0 {
		t.Error("expected non-empty refresh token")
	}
	if res.Data.User.Email != "admin@example.com" {
		t.Errorf("expected user email admin@example.com, got %s", res.Data.User.Email)
	}
	if len(res.Data.User.Roles) == 0 || res.Data.User.Roles[0] != "admin" {
		t.Errorf("expected roles [admin], got %v", res.Data.User.Roles)
	}
}

func TestAuth_LoginWrongPassword(t *testing.T) {
	server, _, _ := setupAuthIntegrationApp(t)

	loginPayload := auth.LoginRequest{
		Identifier: "admin@example.com",
		Password:   "WrongPassword999!",
	}
	body, _ := json.Marshal(loginPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}

	var errRes response.ErrorResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &errRes)

	if errRes.Error.Code != "INVALID_CREDENTIALS" {
		t.Errorf("expected INVALID_CREDENTIALS code, got %s", errRes.Error.Code)
	}
	if errRes.Message != "Identifier atau password tidak valid" {
		t.Errorf("expected generic error message, got %s", errRes.Message)
	}
}

func TestAuth_LoginNonExistentUser(t *testing.T) {
	server, _, _ := setupAuthIntegrationApp(t)

	loginPayload := auth.LoginRequest{
		Identifier: "nobody@example.com",
		Password:   "SomePassword123!",
	}
	body, _ := json.Marshal(loginPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for non-existent user, got %d", rec.Code)
	}

	var errRes response.ErrorResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &errRes)

	if errRes.Error.Code != "INVALID_CREDENTIALS" {
		t.Errorf("expected INVALID_CREDENTIALS code, got %s", errRes.Error.Code)
	}
}

func TestAuth_LoginInactiveUser(t *testing.T) {
	server, _, _ := setupAuthIntegrationApp(t)

	loginPayload := auth.LoginRequest{
		Identifier: "inactive@example.com",
		Password:   "InactiveSecret123!",
	}
	body, _ := json.Marshal(loginPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for inactive user, got %d", rec.Code)
	}

	var errRes response.ErrorResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &errRes)

	if errRes.Error.Code != "INVALID_CREDENTIALS" {
		t.Errorf("expected INVALID_CREDENTIALS, got %s", errRes.Error.Code)
	}
}

func TestUsersMe_ProfileAccess(t *testing.T) {
	server, _, _ := setupAuthIntegrationApp(t)

	// 1. Attempt access without token -> 401
	unauthReq, _ := http.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	unauthRec := httptest.NewRecorder()
	server.Router().ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth token, got %d", unauthRec.Code)
	}

	// 2. Login as student to get token
	loginPayload := auth.LoginRequest{
		Identifier: "student@example.com",
		Password:   "StudentSecret123!",
	}
	body, _ := json.Marshal(loginPayload)
	loginReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	server.Router().ServeHTTP(loginRec, loginReq)

	var loginRes struct {
		Data auth.LoginResponse `json:"data"`
	}
	_ = json.Unmarshal(loginRec.Body.Bytes(), &loginRes)
	token := loginRes.Data.AccessToken

	// 3. Access with valid token -> 200
	authReq, _ := http.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	authReq.Header.Set("Authorization", "Bearer "+token)
	authRec := httptest.NewRecorder()
	server.Router().ServeHTTP(authRec, authReq)

	if authRec.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got %d: %s", authRec.Code, authRec.Body.String())
	}

	var meRes struct {
		Success bool                     `json:"success"`
		Data    auth.UserProfileResponse `json:"data"`
	}
	_ = json.Unmarshal(authRec.Body.Bytes(), &meRes)

	if meRes.Data.Email != "student@example.com" {
		t.Errorf("expected student@example.com, got %s", meRes.Data.Email)
	}
	if meRes.Data.Name != "Budi Santoso" {
		t.Errorf("expected Budi Santoso, got %s", meRes.Data.Name)
	}
}
