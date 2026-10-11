package errors_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	appErrors "educore/internal/shared/errors"
)

func TestAppError_ErrorString(t *testing.T) {
	err := appErrors.New(http.StatusBadRequest, appErrors.CodeBadRequest, "Invalid query parameter")
	expected := "[BAD_REQUEST] Invalid query parameter"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestAppError_Unwrap(t *testing.T) {
	rootErr := errors.New("database connection refused")
	appErr := appErrors.Wrap(rootErr, http.StatusInternalServerError, appErrors.CodeInternalError, "Terjadi kesalahan internal pada server")

	if !errors.Is(appErr, rootErr) {
		t.Errorf("expected errors.Is to match rootErr")
	}

	if appErr.Unwrap() != rootErr {
		t.Errorf("expected Unwrap to return rootErr")
	}
}

func TestFromError(t *testing.T) {
	// Nil error
	if appErrors.FromError(nil) != nil {
		t.Errorf("expected nil for nil input")
	}

	// Already an AppError
	original := appErrors.ErrNotFound("Mahasiswa tidak ditemukan")
	extracted := appErrors.FromError(original)
	if extracted != original {
		t.Errorf("expected same AppError instance")
	}

	// Generic stdlib error
	stdErr := fmt.Errorf("unexpected socket read failure")
	masked := appErrors.FromError(stdErr)
	if masked.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("expected HTTP 500, got %d", masked.HTTPStatus)
	}
	if masked.Code != appErrors.CodeInternalError {
		t.Errorf("expected code INTERNAL_ERROR, got %s", masked.Code)
	}
	if masked.Message != "Terjadi kesalahan internal pada server" {
		t.Errorf("expected masked message, got %s", masked.Message)
	}
	if !errors.Is(masked, stdErr) {
		t.Errorf("expected masked error to wrap original error")
	}
}

func TestStandardConstructors(t *testing.T) {
	tests := []struct {
		name       string
		err        *appErrors.AppError
		wantStatus int
		wantCode   string
	}{
		{"Validation", appErrors.ErrValidation(nil), http.StatusBadRequest, appErrors.CodeValidationError},
		{"BadRequest", appErrors.ErrBadRequest("Bad"), http.StatusBadRequest, appErrors.CodeBadRequest},
		{"Unauthorized", appErrors.ErrUnauthorized(""), http.StatusUnauthorized, appErrors.CodeUnauthorized},
		{"Forbidden", appErrors.ErrForbidden(""), http.StatusForbidden, appErrors.CodeForbidden},
		{"NotFound", appErrors.ErrNotFound(""), http.StatusNotFound, appErrors.CodeResourceNotFound},
		{"RouteNotFound", appErrors.ErrRouteNotFound(), http.StatusNotFound, appErrors.CodeRouteNotFound},
		{"MethodNotAllowed", appErrors.ErrMethodNotAllowed(), http.StatusMethodNotAllowed, appErrors.CodeMethodNotAllowed},
		{"Conflict", appErrors.ErrConflict("Duplicate NIM"), http.StatusConflict, appErrors.CodeConflict},
		{"ContentTooLarge", appErrors.ErrContentTooLarge(""), http.StatusRequestEntityTooLarge, appErrors.CodeContentTooLarge},
		{"UnsupportedMediaType", appErrors.ErrUnsupportedMediaType(""), http.StatusUnsupportedMediaType, appErrors.CodeUnsupportedMediaType},
		{"DependencyUnavailable", appErrors.ErrDependencyUnavailable("", nil), http.StatusServiceUnavailable, appErrors.CodeDependencyUnavailable},
		{"Internal", appErrors.ErrInternal(nil), http.StatusInternalServerError, appErrors.CodeInternalError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.HTTPStatus != tt.wantStatus {
				t.Errorf("HTTPStatus = %d, want %d", tt.err.HTTPStatus, tt.wantStatus)
			}
			if tt.err.Code != tt.wantCode {
				t.Errorf("Code = %s, want %s", tt.err.Code, tt.wantCode)
			}
		})
	}
}
