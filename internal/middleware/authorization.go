package middleware

import (
	"educore/internal/shared/errors"
	"educore/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// RequireRoles returns a Gin middleware that enforces role-based access control.
// The user must possess at least one of the specified allowed roles.
func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := GetRequestID(c)

		principal, exists := GetPrincipal(c)
		if !exists || principal == nil {
			response.RenderFromError(c, errors.ErrUnauthorized("Autentikasi diperlukan sebelum otorisasi role"), reqID)
			c.Abort()
			return
		}

		if !principal.HasAnyRole(allowedRoles...) {
			response.RenderFromError(c, errors.ErrForbidden("Akses ditolak: Anda tidak memiliki role yang diizinkan untuk resource ini"), reqID)
			c.Abort()
			return
		}

		c.Next()
	}
}
