package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"educore/internal/config"
	"educore/internal/modules/auth"
	"educore/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func setupAuthTestEngine(tokenMgr *auth.TokenManager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())

	router.GET("/protected", Authenticate(tokenMgr), func(c *gin.Context) {
		principal, _ := GetPrincipal(c)
		response.OK(c, "Success", gin.H{
			"user_id": principal.UserID.String(),
			"roles":   principal.Roles,
		}, nil)
	})

	return router
}

func TestAuthenticate_MissingHeader(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:         "test-secret-that-is-at-least-32-chars-long!",
		JWTIssuer:         "test-issuer",
		JWTAudience:       "test-audience",
		JWTAccessTokenTTL: 15 * time.Minute,
	}
	tm := auth.NewTokenManager(cfg)
	router := setupAuthTestEngine(tm)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}

	var res response.ErrorResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res.Error.Code != "UNAUTHORIZED" {
		t.Errorf("expected error code UNAUTHORIZED, got %s", res.Error.Code)
	}
}

func TestAuthenticate_InvalidScheme(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:         "test-secret-that-is-at-least-32-chars-long!",
		JWTIssuer:         "test-issuer",
		JWTAudience:       "test-audience",
		JWTAccessTokenTTL: 15 * time.Minute,
	}
	tm := auth.NewTokenManager(cfg)
	router := setupAuthTestEngine(tm)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestAuthenticate_InvalidOrExpiredToken(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:         "test-secret-that-is-at-least-32-chars-long!",
		JWTIssuer:         "test-issuer",
		JWTAudience:       "test-audience",
		JWTAccessTokenTTL: -1 * time.Minute, // Expired immediately
	}
	tm := auth.NewTokenManager(cfg)
	router := setupAuthTestEngine(tm)

	u := &auth.User{
		ID:    uuid.New(),
		Email: "expired@example.com",
		Roles: []string{"admin"},
	}
	expiredToken, _ := tm.GenerateAccessToken(u)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for expired token, got %d", rec.Code)
	}
}

func TestAuthenticate_ValidToken(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:         "test-secret-that-is-at-least-32-chars-long!",
		JWTIssuer:         "test-issuer",
		JWTAudience:       "test-audience",
		JWTAccessTokenTTL: 15 * time.Minute,
	}
	tm := auth.NewTokenManager(cfg)
	router := setupAuthTestEngine(tm)

	testID := uuid.New()
	u := &auth.User{
		ID:    testID,
		Email: "valid@example.com",
		Name:  "Valid User",
		Roles: []string{"dosen"},
	}
	validToken, _ := tm.GenerateAccessToken(u)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res response.Response
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if !res.Success {
		t.Errorf("expected success true, got false")
	}
}
