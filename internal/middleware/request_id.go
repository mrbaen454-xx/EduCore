package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	HeaderXRequestID = "X-Request-ID"
	ContextKeyReqID  = "request_id"
)

type contextKey string

const requestIDContextKey contextKey = "RequestID"

// validRequestIDRegex allows alphanumeric, dashes, and underscores (1 to 64 chars)
var validRequestIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]{1,64}$`)

// RequestID returns a Gin middleware that extracts or generates a unique request ID.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		incomingID := strings.TrimSpace(c.GetHeader(HeaderXRequestID))

		var requestID string
		if incomingID != "" && validRequestIDRegex.MatchString(incomingID) {
			requestID = incomingID
		} else {
			requestID = generateRandomID()
		}

		// Store in Gin context
		c.Set(ContextKeyReqID, requestID)

		// Store in standard context.Context
		ctx := context.WithValue(c.Request.Context(), requestIDContextKey, requestID)
		c.Request = c.Request.WithContext(ctx)

		// Set response header
		c.Header(HeaderXRequestID, requestID)

		c.Next()
	}
}

// GetRequestID retrieves the request ID from gin.Context.
func GetRequestID(c *gin.Context) string {
	if val, exists := c.Get(ContextKeyReqID); exists {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return GetRequestIDFromContext(c.Request.Context())
}

// GetRequestIDFromContext retrieves the request ID from standard context.Context.
func GetRequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(requestIDContextKey).(string); ok {
		return id
	}
	return ""
}

// generateRandomID generates a 32-character hexadecimal cryptographically secure random string.
func generateRandomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback should rand.Read fail
		return "fallback-req-id"
	}
	return hex.EncodeToString(b)
}
