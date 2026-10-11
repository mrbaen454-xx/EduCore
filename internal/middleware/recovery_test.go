package middleware_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"educore/internal/middleware"
	"educore/internal/shared/response"

	"github.com/gin-gonic/gin"
)

func TestRecovery_HandlesPanic(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery(logger))

	r.GET("/panic", func(c *gin.Context) {
		panic("database connection unexpected crash")
	})

	r.GET("/normal", func(c *gin.Context) {
		c.String(http.StatusOK, "all good")
	})

	// 1. Send request that panics
	req := httptest.NewRequest("GET", "/panic", nil)
	req.Header.Set(middleware.HeaderXRequestID, "panic-req-001")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected HTTP 500, got %d", w.Code)
	}

	var res response.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}

	if res.Success != false {
		t.Errorf("expected success: false, got true")
	}
	if res.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("expected code INTERNAL_ERROR, got %s", res.Error.Code)
	}
	if res.Meta == nil || res.Meta.RequestID != "panic-req-001" {
		t.Errorf("expected request_id 'panic-req-001' in meta, got %v", res.Meta)
	}

	// Ensure no stack trace or raw panic message leaked into response body
	bodyStr := w.Body.String()
	if strings.Contains(bodyStr, "database connection unexpected crash") {
		t.Errorf("raw panic message leaked into client response body: %s", bodyStr)
	}
	if strings.Contains(bodyStr, "runtime/debug") || strings.Contains(bodyStr, "goroutine") {
		t.Errorf("stack trace leaked into client response body: %s", bodyStr)
	}

	// Ensure internal logger captured the panic and stack trace
	logStr := logBuf.String()
	if !strings.Contains(logStr, "database connection unexpected crash") {
		t.Errorf("internal logger did not record panic message")
	}
	if !strings.Contains(logStr, "panic-req-001") {
		t.Errorf("internal logger did not record request ID")
	}

	// 2. Ensure subsequent requests are served normally
	req2 := httptest.NewRequest("GET", "/normal", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected subsequent request to succeed with 200, got %d", w2.Code)
	}
	body2, _ := io.ReadAll(w2.Body)
	if string(body2) != "all good" {
		t.Errorf("expected 'all good', got %s", string(body2))
	}
}
