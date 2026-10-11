package auth

import (
	"time"

	"github.com/google/uuid"
)

// User represents the core user account domain entity.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Name         string
	IsActive     bool
	Roles        []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Role represents an authorization role in the system.
type Role struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// RefreshToken represents a persisted session refresh token record.
// Raw tokens are never stored; only cryptographic SHA-256 hashes are persisted.
type RefreshToken struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  string
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	ReplacedBy *uuid.UUID
	UserAgent  string
	ClientIP   string
	CreatedAt  time.Time
}

// IsRevoked checks if the refresh token has been revoked.
func (rt *RefreshToken) IsRevoked() bool {
	return rt.RevokedAt != nil
}

// IsExpired checks if the refresh token has expired.
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// Principal represents the verified identity extracted from an access token.
type Principal struct {
	UserID uuid.UUID
	Email  string
	Name   string
	Roles  []string
}

// HasRole checks if the principal possesses a specific role.
func (p *Principal) HasRole(role string) bool {
	for _, r := range p.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the principal possesses at least one of the specified roles.
func (p *Principal) HasAnyRole(roles ...string) bool {
	for _, required := range roles {
		if p.HasRole(required) {
			return true
		}
	}
	return false
}
