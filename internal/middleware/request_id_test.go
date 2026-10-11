package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"educore/internal/middleware"

	"github.com/gin-gonic/gin"
)

func setupRouterWithRequestID() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/test", func(c *gin.Context) {
		reqID := middleware.GetRequestID(c)
		c.String(http.StatusOK, reqID)
	})
	return r
}

func TestRequestID_MissingHeader(t *testing.T) {
	r := setupRouterWithRequestID()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resHeader := w.Header().Get(middleware.HeaderXRequestID)
	if resHeader == "" {
		t.Fatal("expected X-Request-ID header in response")
	}

	body := w.Body.String()
	if body != resHeader {
		t.Errorf("expected context request ID (%s) to match header (%s)", body, resHeader)
	}

	if len(resHeader) != 32 {
		t.Errorf("expected 32 hex chars, got %d chars: %s", len(resHeader), resHeader)
	}
}

func TestRequestID_ValidHeader(t *testing.T) {
	r := setupRouterWithRequestID()

	customID := "custom-req-id-123_ABC"
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(middleware.HeaderXRequestID, customID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resHeader := w.Header().Get(middleware.HeaderXRequestID)
	if resHeader != customID {
		t.Errorf("expected %s, got %s", customID, resHeader)
	}
	if w.Body.String() != customID {
		t.Errorf("expected body %s, got %s", customID, w.Body.String())
	}
}

func TestRequestID_InvalidCharacters(t *testing.T) {
	r := setupRouterWithRequestID()

	// Control character, newline, and space injection attempts
	invalids := []string{
		"invalid id with spaces",
		"id-with\nnewline",
		"id-with\r\ninjection",
		"id;drop table;",
		"<script>alert(1)</script>",
	}

	for _, invalid := range invalids {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(middleware.HeaderXRequestID, invalid)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		resHeader := w.Header().Get(middleware.HeaderXRequestID)
		if resHeader == invalid {
			t.Errorf("invalid ID %q was accepted without sanitization/replacement", invalid)
		}
		if resHeader == "" {
			t.Errorf("expected newly generated ID, got empty header for input %q", invalid)
		}
	}
}

func TestRequestID_TooLong(t *testing.T) {
	r := setupRouterWithRequestID()

	tooLongID := strings.Repeat("a", 65)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(middleware.HeaderXRequestID, tooLongID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resHeader := w.Header().Get(middleware.HeaderXRequestID)
	if resHeader == tooLongID {
		t.Errorf("expected ID longer than 64 characters to be rejected and replaced")
	}
	if len(resHeader) != 32 {
		t.Errorf("expected newly generated 32-char ID, got %s", resHeader)
	}
}
