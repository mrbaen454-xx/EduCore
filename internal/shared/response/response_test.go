package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"educore/internal/shared/response"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSuccessResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"id": "123"}
	meta := map[string]int{"total": 1}
	response.Success(c, http.StatusOK, "Berhasil memproses data", data, meta)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if body["success"] != true {
		t.Errorf("expected success to be true, got %v", body["success"])
	}
	if body["message"] != "Berhasil memproses data" {
		t.Errorf("unexpected message: %v", body["message"])
	}
	if body["data"] == nil {
		t.Errorf("expected data not to be nil")
	}
	if body["meta"] == nil {
		t.Errorf("expected meta not to be nil")
	}
}

func TestSuccessResponse_NilDataAndMeta(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	response.Success(c, http.StatusOK, "OK", nil, nil)

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	// Should explicitly serialize as null rather than being omitted
	if _, exists := body["data"]; !exists {
		t.Errorf("expected 'data' key to exist in response body")
	}
	if body["data"] != nil {
		t.Errorf("expected 'data' to be null, got %v", body["data"])
	}
	if _, exists := body["meta"]; !exists {
		t.Errorf("expected 'meta' key to exist in response body")
	}
	if body["meta"] != nil {
		t.Errorf("expected 'meta' to be null, got %v", body["meta"])
	}
}

func TestNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	response.NoContent(c)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body for 204, got %s", w.Body.String())
	}
}

func TestErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	details := []response.ErrorDetail{
		{Field: "email", Message: "Format email tidak valid"},
	}
	response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validasi request gagal", details, "req-12345")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var res response.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if res.Success != false {
		t.Errorf("expected success to be false, got %v", res.Success)
	}
	if res.Message != "Validasi request gagal" {
		t.Errorf("unexpected message: %s", res.Message)
	}
	if res.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected error code VALIDATION_ERROR, got %s", res.Error.Code)
	}
	if len(res.Error.Details) != 1 {
		t.Fatalf("expected 1 error detail, got %d", len(res.Error.Details))
	}
	if res.Error.Details[0].Field != "email" {
		t.Errorf("expected field 'email', got %s", res.Error.Details[0].Field)
	}
	if res.Meta == nil || res.Meta.RequestID != "req-12345" {
		t.Errorf("expected request_id 'req-12345', got %v", res.Meta)
	}
}

func TestErrorResponse_EmptyDetails(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	response.Error(c, http.StatusNotFound, "ROUTE_NOT_FOUND", "Endpoint tidak ditemukan", nil, "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	errMap, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object in response")
	}

	// details must serialize to empty array [] not null
	details, ok := errMap["details"].([]any)
	if !ok {
		t.Fatalf("expected details to be JSON array, got %T", errMap["details"])
	}
	if len(details) != 0 {
		t.Errorf("expected empty details array, got length %d", len(details))
	}
	if body["meta"] != nil {
		t.Errorf("expected meta to be null when requestID is empty, got %v", body["meta"])
	}
}
