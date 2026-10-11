package auth

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"educore/internal/config"
	"educore/internal/shared/errors"

	"github.com/google/uuid"
)

// Service defines the business operations for authentication and authorization.
type Service struct {
	repo     *Repository
	hasher   PasswordHasher
	tokenMgr *TokenManager
	cfg      *config.Config
	logger   *slog.Logger

	// dummyHash is used for timing attack mitigation when an identifier does not exist.
	dummyHash string
}

// NewService creates a new Service instance.
func NewService(repo *Repository, hasher PasswordHasher, tokenMgr *TokenManager, cfg *config.Config, logger *slog.Logger) *Service {
	// Pre-generate a dummy Argon2id hash to compare against on invalid user lookups
	dummy, _ := hasher.Hash("dummy-entropy-protection-123456")
	return &Service{
		repo:      repo,
		hasher:    hasher,
		tokenMgr:  tokenMgr,
		cfg:       cfg,
		logger:    logger,
		dummyHash: dummy,
	}
}

// Login authenticates user credentials, checks account status, and issues access & refresh tokens.
func (s *Service) Login(ctx context.Context, req *LoginRequest, clientIP, userAgent string) (*LoginResponse, error) {
	cleanIdentifier := strings.ToLower(strings.TrimSpace(req.Identifier))

	user, err := s.repo.GetUserByEmail(ctx, cleanIdentifier)
	if err != nil {
		s.logger.Error("Database error during user lookup", "error", err)
		return nil, errors.ErrInternal(err)
	}

	// Mitigation against timing attacks and account enumeration:
	// If user does not exist, perform dummy verification to maintain constant computation time.
	if user == nil {
		_, _ = s.hasher.Verify(req.Password, s.dummyHash)
		s.logger.Warn("Failed login attempt: identifier not found", "client_ip", clientIP)
		return nil, errors.ErrInvalidCredentials()
	}

	// Verify candidate password against stored Argon2id hash
	match, err := s.hasher.Verify(req.Password, user.PasswordHash)
	if err != nil || !match {
		s.logger.Warn("Failed login attempt: invalid password", "user_id", user.ID, "client_ip", clientIP)
		return nil, errors.ErrInvalidCredentials()
	}

	// Verify account status
	if !user.IsActive {
		s.logger.Warn("Failed login attempt: account is inactive", "user_id", user.ID, "client_ip", clientIP)
		return nil, errors.ErrInvalidCredentials()
	}

	// Issue JWT access token
	accessToken, err := s.tokenMgr.GenerateAccessToken(user)
	if err != nil {
		s.logger.Error("Failed to issue access token", "user_id", user.ID, "error", err)
		return nil, errors.ErrInternal(err)
	}

	// Generate cryptographically random refresh token
	rawRefreshToken, tokenHash, err := s.tokenMgr.GenerateRefreshToken()
	if err != nil {
		s.logger.Error("Failed to generate refresh token", "user_id", user.ID, "error", err)
		return nil, errors.ErrInternal(err)
	}

	now := time.Now()
	rt := &RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(s.cfg.JWTRefreshTokenTTL),
		UserAgent: userAgent,
		ClientIP:  clientIP,
		CreatedAt: now,
	}

	if err := s.repo.CreateRefreshToken(ctx, rt); err != nil {
		s.logger.Error("Failed to persist refresh token session", "user_id", user.ID, "error", err)
		return nil, errors.ErrInternal(err)
	}

	s.logger.Info("User logged in successfully", "user_id", user.ID, "roles", user.Roles)

	return &LoginResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.cfg.JWTAccessTokenTTL.Seconds()),
		RefreshToken: rawRefreshToken,
		User: UserProfileResponse{
			ID:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			IsActive:  user.IsActive,
			Roles:     user.Roles,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	}, nil
}

// RefreshToken validates an existing refresh token, detects reuse, and performs atomic token rotation.
func (s *Service) RefreshToken(ctx context.Context, rawRefreshToken, clientIP, userAgent string) (*TokenRefreshResponse, error) {
	if len(rawRefreshToken) == 0 {
		return nil, errors.ErrUnauthorized("Refresh token wajib disertakan")
	}

	tokenHash := s.tokenMgr.HashRefreshToken(rawRefreshToken)

	rt, err := s.repo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		s.logger.Error("Database error during refresh token lookup", "error", err)
		return nil, errors.ErrInternal(err)
	}

	if rt == nil {
		s.logger.Warn("Refresh attempt with unknown token", "client_ip", clientIP)
		return nil, errors.ErrUnauthorized("Sesi refresh token tidak valid atau tidak ditemukan")
	}

	// Security: Refresh token reuse detection
	// If a token that was already revoked is used, it indicates potential token theft or breach!
	if rt.IsRevoked() {
		s.logger.Warn("CRITICAL: Refresh token reuse detected! Revoking all sessions for user",
			"user_id", rt.UserID,
			"token_id", rt.ID,
			"client_ip", clientIP,
		)
		_ = s.repo.RevokeAllUserRefreshTokens(ctx, rt.UserID)
		return nil, errors.ErrUnauthorized("Sesi refresh token telah dicabut karena indikasi penggunaan ulang")
	}

	// Check token expiration
	if rt.IsExpired() {
		s.logger.Warn("Refresh attempt with expired token", "token_id", rt.ID, "user_id", rt.UserID)
		return nil, errors.ErrUnauthorized("Sesi refresh token telah kedaluwarsa, silakan login kembali")
	}

	// Verify user account status
	user, err := s.repo.GetUserByID(ctx, rt.UserID)
	if err != nil {
		return nil, errors.ErrInternal(err)
	}
	if user == nil || !user.IsActive {
		s.logger.Warn("Refresh attempt for inactive or non-existent user", "user_id", rt.UserID)
		return nil, errors.ErrUnauthorized("Akun pengguna tidak aktif atau tidak ditemukan")
	}

	// Generate new access token
	newAccessToken, err := s.tokenMgr.GenerateAccessToken(user)
	if err != nil {
		return nil, errors.ErrInternal(err)
	}

	// Generate new refresh token
	newRawRefreshToken, newTokenHash, err := s.tokenMgr.GenerateRefreshToken()
	if err != nil {
		return nil, errors.ErrInternal(err)
	}

	now := time.Now()
	newRT := &RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: newTokenHash,
		ExpiresAt: now.Add(s.cfg.JWTRefreshTokenTTL),
		UserAgent: userAgent,
		ClientIP:  clientIP,
		CreatedAt: now,
	}

	// Atomically rotate: revoke old token and insert new token in a single transaction
	if err := s.repo.RotateRefreshToken(ctx, rt.ID, newRT); err != nil {
		s.logger.Error("Failed to rotate refresh token", "old_token_id", rt.ID, "error", err)
		return nil, errors.ErrInternal(err)
	}

	s.logger.Info("Refresh token rotated successfully", "user_id", user.ID)

	return &TokenRefreshResponse{
		AccessToken:  newAccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.cfg.JWTAccessTokenTTL.Seconds()),
		RefreshToken: newRawRefreshToken,
	}, nil
}

// Logout revokes the provided refresh token session.
func (s *Service) Logout(ctx context.Context, rawRefreshToken string) error {
	if len(rawRefreshToken) == 0 {
		return nil
	}

	tokenHash := s.tokenMgr.HashRefreshToken(rawRefreshToken)
	rt, err := s.repo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		s.logger.Error("Database error during logout token lookup", "error", err)
		return errors.ErrInternal(err)
	}

	if rt != nil && !rt.IsRevoked() {
		if err := s.repo.RevokeRefreshToken(ctx, rt.ID, nil); err != nil {
			s.logger.Error("Failed to revoke refresh token during logout", "token_id", rt.ID, "error", err)
			return errors.ErrInternal(err)
		}
		s.logger.Info("Refresh token session revoked during logout", "token_id", rt.ID, "user_id", rt.UserID)
	}

	return nil
}

// GetProfile retrieves the authenticated user's current profile and roles directly from the authoritative database.
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfileResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Error("Database error fetching user profile", "user_id", userID, "error", err)
		return nil, errors.ErrInternal(err)
	}

	if user == nil {
		return nil, errors.ErrUnauthorized("Pengguna tidak ditemukan")
	}

	if !user.IsActive {
		return nil, errors.ErrUnauthorized("Akun pengguna sedang tidak aktif")
	}

	return &UserProfileResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		IsActive:  user.IsActive,
		Roles:     user.Roles,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}
