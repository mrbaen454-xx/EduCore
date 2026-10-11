package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// Recovery returns a recovery middleware that captures panics,
// logs stack traces via structured logger, and returns a safe HTTP 500 error response.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				reqID := GetRequestID(c)
				stack := string(debug.Stack())

				logger.Error("panic recovered during request execution",
					slog.String("request_id", reqID),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
					slog.Any("panic_error", r),
					slog.String("stack_trace", stack),
				)

				// If response has already been partially written or committed, abort without rewriting body
				if c.Writer.Written() {
					c.Abort()
					return
				}

				internalErr := appErrors.ErrInternal(fmt.Errorf("panic: %v", r))
				response.RenderError(c, internalErr, reqID)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()

		c.Next()
	}
}
