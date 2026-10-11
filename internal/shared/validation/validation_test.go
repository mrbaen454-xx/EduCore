package validation_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/validation"

	"github.com/gin-gonic/gin"
)

type sampleRequest struct {
	Name     string `json:"name" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Role     string `json:"role" validate:"required,oneof=admin dosen mahasiswa"`
	Age      int    `json:"age" validate:"gte=17,lte=100"`
	Semester int    `json:"semester" validate:"min=1,max=14"`
}

func setupTestContext(method, path string, body string, contentType string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	c.Request = req
	return c, w
}

func TestBindJSON_Valid(t *testing.T) {
	body := `{"name":"Budi Santoso","email":"budi@example.com","role":"mahasiswa","age":20,"semester":3}`
	c, _ := setupTestContext("POST", "/test", body, "application/json")

	var req sampleRequest
	err := validation.BindJSON(c, &req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if req.Name != "Budi Santoso" || req.Email != "budi@example.com" || req.Role != "mahasiswa" {
		t.Errorf("decoded struct values mismatched: %+v", req)
	}
}

func TestBindJSON_MissingContentType(t *testing.T) {
	body := `{"name":"Budi Santoso"}`
	c, _ := setupTestContext("POST", "/test", body, "")

	var req sampleRequest
	err := validation.BindJSON(c, &req)
	if err == nil {
		t.Fatalf("expected error for missing Content-Type, got nil")
	}

	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415 Unsupported Media Type, got %v", err)
	}
}

func TestBindJSON_WrongContentType(t *testing.T) {
	body := `name=Budi`
	c, _ := setupTestContext("POST", "/test", body, "application/x-www-form-urlencoded")

	var req sampleRequest
	err := validation.BindJSON(c, &req)
	if err == nil {
		t.Fatalf("expected error for wrong Content-Type, got nil")
	}

	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415 Unsupported Media Type, got %v", err)
	}
}

func TestBindJSON_EmptyBody(t *testing.T) {
	body := `   `
	c, _ := setupTestContext("POST", "/test", body, "application/json")

	var req sampleRequest
	err := validation.BindJSON(c, &req)
	if err == nil {
		t.Fatalf("expected error for empty body, got nil")
	}

	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %v", err)
	}
}

func TestBindJSON_MalformedJSON(t *testing.T) {
	body := `{"name":"Budi Santoso", broken_json`
	c, _ := setupTestContext("POST", "/test", body, "application/json")

	var req sampleRequest
	err := validation.BindJSON(c, &req)
	if err == nil {
		t.Fatalf("expected error for malformed JSON, got nil")
	}

	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %v", err)
	}
}

func TestBindJSON_UnknownField(t *testing.T) {
	body := `{"name":"Budi Santoso","email":"budi@example.com","role":"mahasiswa","age":20,"semester":3,"extra_field":"disallowed"}`
	c, _ := setupTestContext("POST", "/test", body, "application/json")

	var req sampleRequest
	err := validation.BindJSON(c, &req)
	if err == nil {
		t.Fatalf("expected error for unknown field, got nil")
	}

	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for unknown field, got %v", err)
	}
}

func TestBindJSON_MultipleJSONDocuments(t *testing.T) {
	body := `{"name":"Budi Santoso","email":"budi@example.com","role":"mahasiswa","age":20,"semester":3}{"another":"object"}`
	c, _ := setupTestContext("POST", "/test", body, "application/json")

	var req sampleRequest
	err := validation.BindJSON(c, &req)
	if err == nil {
		t.Fatalf("expected error for multiple JSON documents, got nil")
	}

	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %v", err)
	}
}

func TestBindJSON_FieldValidationFailures(t *testing.T) {
	// Name too short, invalid email, invalid role, age < 17, semester > 14
	body := `{"name":"Al","email":"invalid-email","role":"superadmin","age":15,"semester":20}`
	c, _ := setupTestContext("POST", "/test", body, "application/json")

	var req sampleRequest
	err := validation.BindJSON(c, &req)
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}

	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %v", err)
	}

	if appErr.HTTPStatus != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", appErr.HTTPStatus)
	}
	if appErr.Code != appErrors.CodeValidationError {
		t.Errorf("expected code VALIDATION_ERROR, got %s", appErr.Code)
	}
	if len(appErr.Details) != 5 {
		t.Errorf("expected 5 validation error details, got %d", len(appErr.Details))
	}
}
