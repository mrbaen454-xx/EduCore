package auth

import (
	"errors"
	"testing"
	"time"

	"educore/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func newTestTokenConfig() *config.Config {
	return &config.Config{
		JWTSecret:         "test-secret-key-that-is-at-least-32-bytes-long!",
		JWTIssuer:         "test-issuer",
		JWTAudience:       "test-audience",
		JWTAccessTokenTTL: 15 * time.Minute,
	}
}

func TestTokenManager_GenerateAndVerify(t *testing.T) {
	cfg := newTestTokenConfig()
	tm := NewTokenManager(cfg)

	user := &User{
		ID:    uuid.New(),
		Email: "student@example.com",
		Name:  "Mahasiswa Uji",
		Roles: []string{"mahasiswa"},
	}

	tokenStr, err := tm.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("expected no error generating token, got %v", err)
	}
	if len(tokenStr) == 0 {
		t.Fatal("expected non-empty token string")
	}

	principal, err := tm.VerifyAccessToken(tokenStr)
	if err != nil {
		t.Fatalf("expected valid token, got error: %v", err)
	}

	if principal.UserID != user.ID {
		t.Errorf("expected UserID %s, got %s", user.ID, principal.UserID)
	}
	if principal.Email != user.Email {
		t.Errorf("expected Email %s, got %s", user.Email, principal.Email)
	}
	if principal.Name != user.Name {
		t.Errorf("expected Name %s, got %s", user.Name, principal.Name)
	}
	if !principal.HasRole("mahasiswa") {
		t.Error("expected principal to have role 'mahasiswa'")
	}
	if principal.HasRole("admin") {
		t.Error("expected principal to NOT have role 'admin'")
	}
}

func TestTokenManager_ExpiredToken(t *testing.T) {
	cfg := newTestTokenConfig()
	cfg.JWTAccessTokenTTL = -1 * time.Minute // Expired immediately
	tm := NewTokenManager(cfg)

	user := &User{
		ID:    uuid.New(),
		Email: "expired@example.com",
		Name:  "Expired User",
		Roles: []string{"mahasiswa"},
	}

	tokenStr, err := tm.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	_, err = tm.VerifyAccessToken(tokenStr)
	if !errors.Is(err, ErrExpiredToken) {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestTokenManager_InvalidSignature(t *testing.T) {
	cfg1 := newTestTokenConfig()
	tm1 := NewTokenManager(cfg1)

	cfg2 := newTestTokenConfig()
	cfg2.JWTSecret = "different-secret-key-also-at-least-32-chars-long!"
	tm2 := NewTokenManager(cfg2)

	user := &User{
		ID:    uuid.New(),
		Email: "tampered@example.com",
		Name:  "Tampered User",
		Roles: []string{"admin"},
	}

	tokenStr, err := tm1.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	// Verify with a different manager / secret
	_, err = tm2.VerifyAccessToken(tokenStr)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestTokenManager_RejectNoneAlgorithm(t *testing.T) {
	cfg := newTestTokenConfig()
	tm := NewTokenManager(cfg)

	// Craft an unsigned token with 'none' algorithm
	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.New().String(),
			Issuer:    cfg.JWTIssuer,
			Audience:  jwt.ClaimStrings{cfg.JWTAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
		},
		Roles: []string{"admin"},
	}
	noneToken := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, _ := noneToken.SignedString(jwt.UnsafeAllowNoneSignatureType)

	_, err := tm.VerifyAccessToken(tokenStr)
	if err == nil {
		t.Fatal("expected error rejecting 'none' algorithm, got nil")
	}
}

func TestTokenManager_IssuerAndAudienceMismatch(t *testing.T) {
	cfg := newTestTokenConfig()
	tm := NewTokenManager(cfg)

	mismatchCfg := newTestTokenConfig()
	mismatchCfg.JWTIssuer = "rogue-issuer"
	tmRogue := NewTokenManager(mismatchCfg)

	user := &User{
		ID:    uuid.New(),
		Email: "mismatch@example.com",
		Name:  "Mismatch User",
	}

	tokenStr, _ := tmRogue.GenerateAccessToken(user)
	_, err := tm.VerifyAccessToken(tokenStr)
	if err == nil {
		t.Fatal("expected error on issuer mismatch, got nil")
	}
}

func TestTokenManager_RefreshTokenGenerationAndHash(t *testing.T) {
	cfg := newTestTokenConfig()
	tm := NewTokenManager(cfg)

	raw1, hash1, err := tm.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(raw1) == 0 || len(hash1) != 64 {
		t.Fatalf("unexpected raw token or hash length (raw: %d, hash: %d)", len(raw1), len(hash1))
	}

	// Hashing raw token directly produces identical hash
	computedHash := tm.HashRefreshToken(raw1)
	if computedHash != hash1 {
		t.Errorf("expected hash %s, got %s", hash1, computedHash)
	}

	// Different calls produce different random tokens
	raw2, hash2, _ := tm.GenerateRefreshToken()
	if raw1 == raw2 || hash1 == hash2 {
		t.Error("two refresh tokens generated sequentially should be unique")
	}
}
