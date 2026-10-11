package auth

import (
	"log/slog"

	"educore/internal/shared/errors"
	"educore/internal/shared/response"
	"educore/internal/shared/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for authentication and authorization endpoints.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler creates a new Handler instance.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(c *gin.Context) {
	reqID := c.GetString("request_id")

	var req LoginRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	res, err := h.service.Login(c.Request.Context(), &req, clientIP, userAgent)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Login berhasil", res, nil)
}

// RefreshToken handles POST /api/v1/auth/refresh.
func (h *Handler) RefreshToken(c *gin.Context) {
	reqID := c.GetString("request_id")

	var req RefreshTokenRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	res, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken, clientIP, userAgent)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Token berhasil diperbarui", res, nil)
}

// Logout handles POST /api/v1/auth/logout.
func (h *Handler) Logout(c *gin.Context) {
	reqID := c.GetString("request_id")

	var req LogoutRequest
	// Optional body parsing: if body is empty or malformed, continue gracefully
	_ = c.ShouldBindJSON(&req)

	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Logout berhasil", nil, nil)
}

// GetProfile handles GET /api/v1/users/me.
func (h *Handler) GetProfile(c *gin.Context) {
	reqID := c.GetString("request_id")

	val, exists := c.Get("auth_user_id")
	if !exists {
		response.RenderFromError(c, errors.ErrUnauthorized("Identitas pengguna tidak ditemukan pada sesi"), reqID)
		return
	}

	userID, ok := val.(uuid.UUID)
	if !ok {
		response.RenderFromError(c, errors.ErrUnauthorized("Format identitas sesi tidak valid"), reqID)
		return
	}

	profile, err := h.service.GetProfile(c.Request.Context(), userID)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Profil pengguna berhasil diambil", profile, nil)
}

// TestAdminOnly handles GET /api/v1/test/admin-only for verifying role authorization.
func (h *Handler) TestAdminOnly(c *gin.Context) {
	response.OK(c, "Akses khusus admin berhasil", gin.H{"access": "granted", "role": "admin"}, nil)
}
