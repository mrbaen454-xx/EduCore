package response

import (
	"net/http"

	"educore/internal/shared/errors"

	"github.com/gin-gonic/gin"
)

// Response represents the standard success response envelope.
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	Meta    any    `json:"meta"`
}

// ErrorDetail represents a specific error detail (e.g. validation error per field).
type ErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// ErrorBody contains the machine-readable code and detailed errors.
type ErrorBody struct {
	Code    string        `json:"code"`
	Details []ErrorDetail `json:"details"`
}

// ErrorMeta holds metadata related to an error response, such as tracing ID.
type ErrorMeta struct {
	RequestID string `json:"request_id,omitempty"`
}

// ErrorResponse represents the standard error response envelope.
type ErrorResponse struct {
	Success bool       `json:"success"`
	Message string     `json:"message"`
	Error   ErrorBody  `json:"error"`
	Meta    *ErrorMeta `json:"meta"`
}

// Success sends a standardized JSON success response.
func Success(c *gin.Context, httpStatus int, message string, data any, meta any) {
	c.Header("Content-Type", "application/json")
	c.JSON(httpStatus, Response{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// OK sends a 200 OK success response.
func OK(c *gin.Context, message string, data any, meta any) {
	Success(c, http.StatusOK, message, data, meta)
}

// Created sends a 201 Created success response.
func Created(c *gin.Context, message string, data any, meta any) {
	Success(c, http.StatusCreated, message, data, meta)
}

// NoContent sends a 204 No Content response without body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
}

// Error sends a standardized JSON error response.
func Error(c *gin.Context, httpStatus int, code string, message string, details []ErrorDetail, requestID string) {
	c.Header("Content-Type", "application/json")

	if details == nil {
		details = []ErrorDetail{}
	}

	var meta *ErrorMeta
	if requestID != "" {
		meta = &ErrorMeta{
			RequestID: requestID,
		}
	}

	c.JSON(httpStatus, ErrorResponse{
		Success: false,
		Message: message,
		Error: ErrorBody{
			Code:    code,
			Details: details,
		},
		Meta: meta,
	})
}

// RenderError sends a standardized JSON error response based on an AppError.
func RenderError(c *gin.Context, appErr *errors.AppError, requestID string) {
	if appErr == nil {
		appErr = errors.ErrInternal(nil)
	}

	details := make([]ErrorDetail, len(appErr.Details))
	for i, d := range appErr.Details {
		details[i] = ErrorDetail{
			Field:   d.Field,
			Message: d.Message,
		}
	}

	Error(c, appErr.HTTPStatus, appErr.Code, appErr.Message, details, requestID)
}

// RenderFromError inspects any Go error, converts it via errors.FromError, and sends the standard response.
func RenderFromError(c *gin.Context, err error, requestID string) {
	appErr := errors.FromError(err)
	RenderError(c, appErr, requestID)
}
