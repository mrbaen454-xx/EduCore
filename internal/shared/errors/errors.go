package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Standard error codes according to EduCore REST API standard
const (
	CodeValidationError       = "VALIDATION_ERROR"
	CodeBadRequest            = "BAD_REQUEST"
	CodeUnauthorized          = "UNAUTHORIZED"
	CodeForbidden             = "FORBIDDEN"
	CodeResourceNotFound      = "RESOURCE_NOT_FOUND"
	CodeRouteNotFound         = "ROUTE_NOT_FOUND"
	CodeMethodNotAllowed      = "METHOD_NOT_ALLOWED"
	CodeConflict              = "CONFLICT"
	CodeContentTooLarge       = "CONTENT_TOO_LARGE"
	CodeUnsupportedMediaType  = "UNSUPPORTED_MEDIA_TYPE"
	CodeDependencyUnavailable = "DEPENDENCY_UNAVAILABLE"
	CodeInternalError         = "INTERNAL_ERROR"
)

// Detail represents a granular field-level or contextual error detail.
type Detail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// AppError represents a structured, domain-aware application error.
type AppError struct {
	HTTPStatus  int
	Code        string
	Message     string
	Details     []Detail
	InternalErr error
}

// Error implements the standard Go error interface.
func (e *AppError) Error() string {
	if e.InternalErr != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.InternalErr)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap supports Go 1.13+ error unwrapping for internal root causes.
func (e *AppError) Unwrap() error {
	return e.InternalErr
}

// WithDetails attaches field-level or contextual details to an AppError.
func (e *AppError) WithDetails(details []Detail) *AppError {
	e.Details = details
	return e
}

// New creates a new client-facing AppError.
func New(status int, code string, message string) *AppError {
	return &AppError{
		HTTPStatus: status,
		Code:       code,
		Message:    message,
		Details:    make([]Detail, 0),
	}
}

// Wrap creates an AppError wrapping an underlying internal error.
func Wrap(err error, status int, code string, message string) *AppError {
	return &AppError{
		HTTPStatus:  status,
		Code:        code,
		Message:     message,
		Details:     make([]Detail, 0),
		InternalErr: err,
	}
}

// ErrValidation returns an AppError for input validation failures (HTTP 400).
func ErrValidation(details []Detail) *AppError {
	return &AppError{
		HTTPStatus: http.StatusBadRequest,
		Code:       CodeValidationError,
		Message:    "Validasi request gagal",
		Details:    details,
	}
}

// ErrBadRequest returns an AppError for general malformed requests (HTTP 400).
func ErrBadRequest(message string) *AppError {
	return New(http.StatusBadRequest, CodeBadRequest, message)
}

// ErrUnauthorized returns an AppError when authentication is missing or invalid (HTTP 401).
func ErrUnauthorized(message string) *AppError {
	if message == "" {
		message = "Autentikasi diperlukan untuk mengakses resource ini"
	}
	return New(http.StatusUnauthorized, CodeUnauthorized, message)
}

// ErrForbidden returns an AppError when user has no permission (HTTP 403).
func ErrForbidden(message string) *AppError {
	if message == "" {
		message = "Akses ditolak: Anda tidak memiliki izin untuk resource ini"
	}
	return New(http.StatusForbidden, CodeForbidden, message)
}

// ErrNotFound returns an AppError for missing resources (HTTP 404).
func ErrNotFound(message string) *AppError {
	if message == "" {
		message = "Resource yang diminta tidak ditemukan"
	}
	return New(http.StatusNotFound, CodeResourceNotFound, message)
}

// ErrRouteNotFound returns an AppError for unregistered routes (HTTP 404).
func ErrRouteNotFound() *AppError {
	return New(http.StatusNotFound, CodeRouteNotFound, "Endpoint tidak ditemukan")
}

// ErrMethodNotAllowed returns an AppError for unsupported HTTP methods (HTTP 405).
func ErrMethodNotAllowed() *AppError {
	return New(http.StatusMethodNotAllowed, CodeMethodNotAllowed, "Metode HTTP tidak didukung untuk endpoint ini")
}

// ErrConflict returns an AppError for state or unique constraint conflicts (HTTP 409).
func ErrConflict(message string) *AppError {
	return New(http.StatusConflict, CodeConflict, message)
}

// ErrContentTooLarge returns an AppError when request payload exceeds limits (HTTP 413).
func ErrContentTooLarge(message string) *AppError {
	if message == "" {
		message = "Ukuran body request melebihi batas maksimum yang diizinkan"
	}
	return New(http.StatusRequestEntityTooLarge, CodeContentTooLarge, message)
}

// ErrUnsupportedMediaType returns an AppError for unsupported Content-Type (HTTP 415).
func ErrUnsupportedMediaType(message string) *AppError {
	if message == "" {
		message = "Content-Type request tidak didukung, gunakan application/json"
	}
	return New(http.StatusUnsupportedMediaType, CodeUnsupportedMediaType, message)
}

// ErrDependencyUnavailable returns an AppError when a critical dependency is down (HTTP 503).
func ErrDependencyUnavailable(message string, err error) *AppError {
	if message == "" {
		message = "Layanan dependensi sedang tidak tersedia"
	}
	return Wrap(err, http.StatusServiceUnavailable, CodeDependencyUnavailable, message)
}

// ErrInternal creates a safe, generic internal server error masking sensitive technical details (HTTP 500).
func ErrInternal(err error) *AppError {
	return Wrap(err, http.StatusInternalServerError, CodeInternalError, "Terjadi kesalahan internal pada server")
}

// FromError extracts an *AppError from any error or wraps it as an ErrInternal.
func FromError(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return ErrInternal(err)
}
