package middleware

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger returns a structured logging middleware using log/slog.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := sanitizeLogValue(c.Request.URL.Path)
		rawQuery := sanitizeLogValue(c.Request.URL.RawQuery)

		// Process request
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		reqID := GetRequestID(c)
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}

		attrs := []slog.Attr{
			slog.String("request_id", reqID),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("route", route),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.Int("bytes", c.Writer.Size()),
			slog.String("client_ip", c.ClientIP()),
		}

		if rawQuery != "" {
			// Redact potential sensitive tokens or passwords in query string
			sanitizedQuery := redactSensitiveQuery(rawQuery)
			attrs = append(attrs, slog.String("query", sanitizedQuery))
		}

		msg := "HTTP request completed"

		// Pick level based on HTTP status code
		switch {
		case status >= 500:
			logger.LogAttrs(c.Request.Context(), slog.LevelError, msg, attrs...)
		case status >= 400:
			logger.LogAttrs(c.Request.Context(), slog.LevelWarn, msg, attrs...)
		default:
			logger.LogAttrs(c.Request.Context(), slog.LevelInfo, msg, attrs...)
		}
	}
}

// sanitizeLogValue removes control characters like newlines to prevent log injection.
func sanitizeLogValue(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

// redactSensitiveQuery redacts query values for keys like token, password, secret, key.
func redactSensitiveQuery(query string) string {
	parts := strings.Split(query, "&")
	for i, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			kLower := strings.ToLower(kv[0])
			if strings.Contains(kLower, "token") ||
				strings.Contains(kLower, "pass") ||
				strings.Contains(kLower, "secret") ||
				strings.Contains(kLower, "key") {
				parts[i] = kv[0] + "=[REDACTED]"
			}
		}
	}
	return strings.Join(parts, "&")
}
