package auth

import (
	"time"

	"github.com/google/uuid"
)

// LoginRequest defines the request body for POST /api/v1/auth/login.
type LoginRequest struct {
	Identifier string `json:"identifier" validate:"required,min=3,max=255"`
	Password   string `json:"password" validate:"required,min=8,max=128"`
}

// UserProfileResponse defines the public user representation in responses.
type UserProfileResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LoginResponse defines the response data for successful login.
type LoginResponse struct {
	AccessToken  string              `json:"access_token"`
	TokenType    string              `json:"token_type"`
	ExpiresIn    int64               `json:"expires_in"` // in seconds
	RefreshToken string              `json:"refresh_token"`
	User         UserProfileResponse `json:"user"`
}

// RefreshTokenRequest defines the request body for POST /api/v1/auth/refresh.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// TokenRefreshResponse defines the response data for successful token refresh.
type TokenRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"` // in seconds
	RefreshToken string `json:"refresh_token"`
}

// LogoutRequest defines the request body for POST /api/v1/auth/logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}
