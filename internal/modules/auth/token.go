package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"educore/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken       = errors.New("invalid or malformed access token")
	ErrExpiredToken       = errors.New("access token has expired")
	ErrInvalidSignature   = errors.New("access token signature is invalid")
	ErrUnexpectedAlg      = errors.New("unexpected token signing algorithm")
	ErrInvalidTokenClaims = errors.New("token claims validation failed")
	ErrEmptyToken         = errors.New("token string cannot be empty")
)

// CustomClaims defines the JWT claims structure for EduCore access tokens.
type CustomClaims struct {
	jwt.RegisteredClaims
	Roles []string `json:"roles"`
	Email string   `json:"email"`
	Name  string   `json:"name"`
}

// TokenManager handles generation, parsing, and verification of access and refresh tokens.
type TokenManager struct {
	secret         []byte
	issuer         string
	audience       string
	accessTokenTTL time.Duration
}

// NewTokenManager creates a new TokenManager instance from application configuration.
func NewTokenManager(cfg *config.Config) *TokenManager {
	secret := []byte(cfg.JWTSecret)
	if len(secret) < 32 {
		secret = []byte("default-dev-secret-that-is-at-least-32-bytes-long!")
	}
	issuer := cfg.JWTIssuer
	if issuer == "" {
		issuer = "educore-api"
	}
	audience := cfg.JWTAudience
	if audience == "" {
		audience = "educore-app"
	}
	ttl := cfg.JWTAccessTokenTTL
	if ttl == 0 {
		ttl = 15 * time.Minute
	}
	return &TokenManager{
		secret:         secret,
		issuer:         issuer,
		audience:       audience,
		accessTokenTTL: ttl,
	}
}

// GenerateAccessToken signs a new JWT access token for the given user.
func (tm *TokenManager) GenerateAccessToken(user *User) (string, error) {
	now := time.Now()
	tokenID := uuid.New().String()

	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			Issuer:    tm.issuer,
			Audience:  jwt.ClaimStrings{tm.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.accessTokenTTL)),
			ID:        tokenID,
		},
		Roles: user.Roles,
		Email: user.Email,
		Name:  user.Name,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(tm.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return signed, nil
}

// VerifyAccessToken verifies signature, algorithm, issuer, audience, and expiration of a JWT access token.
func (tm *TokenManager) VerifyAccessToken(tokenStr string) (*Principal, error) {
	if len(tokenStr) == 0 {
		return nil, ErrEmptyToken
	}

	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Enforce HS256 algorithm strictly; reject 'none' or asymmetric algorithms
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("%w: expected %s, got %s", ErrUnexpectedAlg, jwt.SigningMethodHS256.Alg(), token.Method.Alg())
		}
		return tm.secret, nil
	}, jwt.WithIssuer(tm.issuer), jwt.WithAudience(tm.audience))

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, ErrInvalidSignature
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidTokenClaims
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user id subject", ErrInvalidTokenClaims)
	}

	return &Principal{
		UserID: userID,
		Email:  claims.Email,
		Name:   claims.Name,
		Roles:  claims.Roles,
	}, nil
}

// GenerateRefreshToken generates 32 cryptographically secure random bytes as raw token
// and its corresponding SHA-256 hex hash for persistent database storage.
func (tm *TokenManager) GenerateRefreshToken() (rawToken string, tokenHash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes for refresh token: %w", err)
	}

	rawToken = base64.RawURLEncoding.EncodeToString(b)
	tokenHash = tm.HashRefreshToken(rawToken)
	return rawToken, tokenHash, nil
}

// HashRefreshToken calculates the SHA-256 hex hash of a raw refresh token string.
func (tm *TokenManager) HashRefreshToken(rawToken string) string {
	h := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(h[:])
}
