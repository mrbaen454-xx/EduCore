package middleware_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"educore/internal/middleware"
	"educore/internal/shared/response"
	"educore/internal/shared/validation"

	"github.com/gin-gonic/gin"
)

type smallPayload struct {
	Text string `json:"text" validate:"required"`
}

func TestBodyLimit_Exceeded(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestID())
	// Limit to 100 bytes
	r.Use(middleware.BodyLimit(100))

	r.POST("/upload", func(c *gin.Context) {
		var p smallPayload
		if err := validation.BindJSON(c, &p); err != nil {
			response.RenderFromError(c, err, middleware.GetRequestID(c))
			return
		}
		response.OK(c, "success", p, nil)
	})

	// 200 bytes string
	largeBody := `{"text":"` + strings.Repeat("A", 200) + `"}`
	req := httptest.NewRequest("POST", "/upload", bytes.NewBufferString(largeBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected HTTP 413, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestBodyLimit_Allowed(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.BodyLimit(1000))

	r.POST("/upload", func(c *gin.Context) {
		var p smallPayload
		if err := validation.BindJSON(c, &p); err != nil {
			response.RenderFromError(c, err, middleware.GetRequestID(c))
			return
		}
		response.OK(c, "success", p, nil)
	})

	smallBody := `{"text":"Hello World"}`
	req := httptest.NewRequest("POST", "/upload", bytes.NewBufferString(smallBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", w.Code)
	}
}
