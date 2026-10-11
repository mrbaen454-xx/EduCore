package middleware

import (
	"strings"

	"educore/internal/modules/auth"
	"educore/internal/shared/errors"
	"educore/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	ContextKeyPrincipal = "auth_principal"
	ContextKeyUserID    = "auth_user_id"
	BearerPrefix        = "Bearer "
)

// Authenticate returns a Gin middleware that extracts and validates the JWT Bearer access token.
func Authenticate(tokenMgr *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := GetRequestID(c)

		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			response.RenderFromError(c, errors.ErrUnauthorized("Header Authorization dengan skema Bearer diperlukan"), reqID)
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, BearerPrefix) {
			response.RenderFromError(c, errors.ErrUnauthorized("Format header Authorization tidak valid, gunakan format 'Bearer <token>'"), reqID)
			c.Abort()
			return
		}

		tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, BearerPrefix))
		if tokenStr == "" {
			response.RenderFromError(c, errors.ErrUnauthorized("Token akses tidak boleh kosong"), reqID)
			c.Abort()
			return
		}

		principal, err := tokenMgr.VerifyAccessToken(tokenStr)
		if err != nil {
			response.RenderFromError(c, errors.ErrUnauthorized("Token akses tidak valid atau telah kedaluwarsa"), reqID)
			c.Abort()
			return
		}

		// Store verified principal and user id in context
		c.Set(ContextKeyPrincipal, principal)
		c.Set(ContextKeyUserID, principal.UserID)

		c.Next()
	}
}

// GetPrincipal retrieves the authenticated Principal from the Gin context.
func GetPrincipal(c *gin.Context) (*auth.Principal, bool) {
	val, exists := c.Get(ContextKeyPrincipal)
	if !exists {
		return nil, false
	}
	principal, ok := val.(*auth.Principal)
	return principal, ok
}

// GetUserID retrieves the authenticated UserID from the Gin context.
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get(ContextKeyUserID)
	if !exists {
		return uuid.Nil, false
	}
	id, ok := val.(uuid.UUID)
	return id, ok
}
