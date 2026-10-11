package integration_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"educore/internal/config"
	"educore/internal/middleware"
	"educore/internal/platform/httpserver"
	"educore/internal/shared/response"

	"github.com/goccy/go-yaml"
)

func newTestServer() *httpserver.Server {
	cfg := &config.Config{
		AppEnv:              "testing",
		AppPort:             8080,
		AppTimezone:         "Asia/Jakarta",
		LogLevel:            "error",
		HTTPReadTimeout:     5 * time.Second,
		HTTPWriteTimeout:    5 * time.Second,
		HTTPIdleTimeout:     10 * time.Second,
		HTTPShutdownTimeout: 5 * time.Second,
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	return httpserver.New(cfg, logger)
}

func TestHealthLiveEndpoint(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected application/json, got %s", contentType)
	}

	reqID := w.Header().Get(middleware.HeaderXRequestID)
	if reqID == "" {
		t.Errorf("expected X-Request-ID header in response")
	}

	var res response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if !res.Success {
		t.Errorf("expected success: true, got false")
	}

	dataMap, ok := res.Data.(map[string]any)
	if !ok || dataMap["status"] != "up" {
		t.Errorf("expected data.status = 'up', got %v", res.Data)
	}
}

func TestHealthReadyEndpoint(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", w.Code)
	}

	var res response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if !res.Success {
		t.Errorf("expected success: true, got false")
	}

	dataMap, ok := res.Data.(map[string]any)
	if !ok || dataMap["status"] != "ready" {
		t.Errorf("expected data.status = 'ready', got %v", res.Data)
	}
}

func TestAPIV1HealthEndpoint(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", w.Code)
	}

	var res response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if !res.Success {
		t.Errorf("expected success: true, got false")
	}

	dataMap, ok := res.Data.(map[string]any)
	if !ok || dataMap["version"] != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %v", res.Data)
	}
}

func TestRouteNotFound_JSON404(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest("GET", "/api/v1/non-existent-endpoint", nil)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected HTTP 404, got %d", w.Code)
	}

	var errRes response.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errRes); err != nil {
		t.Fatalf("failed to parse 404 JSON error response: %v", err)
	}

	if errRes.Success {
		t.Errorf("expected success: false, got true")
	}
	if errRes.Error.Code != "ROUTE_NOT_FOUND" {
		t.Errorf("expected code 'ROUTE_NOT_FOUND', got %s", errRes.Error.Code)
	}
	if errRes.Meta == nil || errRes.Meta.RequestID == "" {
		t.Errorf("expected meta.request_id to be populated in 404 response")
	}
}

func TestMethodNotAllowed_JSON405(t *testing.T) {
	server := newTestServer()

	// DELETE on /api/v1/health should return 405 Method Not Allowed
	req := httptest.NewRequest("DELETE", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected HTTP 405 Method Not Allowed, got %d", w.Code)
	}

	var errRes response.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errRes); err != nil {
		t.Fatalf("failed to parse 405 JSON error response: %v", err)
	}

	if errRes.Success {
		t.Errorf("expected success: false, got true")
	}
	if errRes.Error.Code != "METHOD_NOT_ALLOWED" {
		t.Errorf("expected code 'METHOD_NOT_ALLOWED', got %s", errRes.Error.Code)
	}
	if errRes.Meta == nil || errRes.Meta.RequestID == "" {
		t.Errorf("expected meta.request_id to be populated in 405 response")
	}
}

func TestCustomRequestIDPropagation(t *testing.T) {
	server := newTestServer()

	customReqID := "custom-trace-siakad-999"
	req := httptest.NewRequest("GET", "/health/live", nil)
	req.Header.Set(middleware.HeaderXRequestID, customReqID)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", w.Code)
	}

	resReqID := w.Header().Get(middleware.HeaderXRequestID)
	if resReqID != customReqID {
		t.Errorf("expected response header X-Request-ID = %q, got %q", customReqID, resReqID)
	}
}

func TestOpenAPISpecificationValid(t *testing.T) {
	data, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatalf("failed to read openapi.yaml: %v", err)
	}

	var parsed map[string]any
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse openapi.yaml: %v", err)
	}

	if parsed["openapi"] != "3.0.3" {
		t.Errorf("expected openapi version 3.0.3, got %v", parsed["openapi"])
	}

	paths, ok := parsed["paths"].(map[string]any)
	if !ok {
		t.Fatalf("expected paths object in openapi.yaml")
	}

	expectedPaths := []string{"/health/live", "/health/ready", "/api/v1/health"}
	for _, p := range expectedPaths {
		if _, exists := paths[p]; !exists {
			t.Errorf("expected path %q to exist in openapi.yaml", p)
		}
	}
}
