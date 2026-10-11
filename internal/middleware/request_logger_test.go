package middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"educore/internal/middleware"

	"github.com/gin-gonic/gin"
)

func TestRequestLogger(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestLogger(logger))

	r.GET("/api/v1/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/api/v1/test?token=secret_123&page=1", nil)
	req.Header.Set("Authorization", "Bearer sensitive-token-xyz")
	req.Header.Set(middleware.HeaderXRequestID, "test-log-req-id")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	logOutput := logBuf.String()

	// Check that required fields are present in log
	if !strings.Contains(logOutput, `"request_id":"test-log-req-id"`) {
		t.Errorf("expected log to contain request_id, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, `"method":"GET"`) {
		t.Errorf("expected log to contain method GET, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, `"status":200`) {
		t.Errorf("expected log to contain status 200, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, `"latency":`) {
		t.Errorf("expected log to contain latency, got: %s", logOutput)
	}

	// Verify sensitive authorization header is NOT logged
	if strings.Contains(logOutput, "sensitive-token-xyz") {
		t.Errorf("sensitive authorization header was leaked into log: %s", logOutput)
	}

	// Verify query secret was redacted
	if strings.Contains(logOutput, "secret_123") {
		t.Errorf("query token value was not redacted in log: %s", logOutput)
	}
	if !strings.Contains(logOutput, "token=[REDACTED]") {
		t.Errorf("expected redacted query token in log, got: %s", logOutput)
	}
}

func TestRequestLogger_SanitizesNewlines(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestLogger(logger))

	r.GET("/test/*any", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Inject newlines in URL
	req := httptest.NewRequest("GET", "/test/foo%0Abar?param=val%0Dnewline", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	logOutput := logBuf.String()
	// Count newline characters in log output - should only be one ending newline for the log line
	newlineCount := strings.Count(logOutput, "\n")
	if newlineCount > 1 {
		t.Errorf("expected single log line without injected newlines, got count %d in: %s", newlineCount, logOutput)
	}
}
