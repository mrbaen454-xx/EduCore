package faculties

import (
	"log/slog"
	"strconv"
	"strings"

	"educore/internal/middleware"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"
	"educore/internal/shared/response"
	"educore/internal/shared/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for faculty endpoints.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler instantiates a new faculty HTTP handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// Create handles POST /api/v1/faculties
func (h *Handler) Create(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	var req CreateFacultyRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	result, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.Created(c, "Fakultas berhasil dibuat", result, nil)
}

// GetByID handles GET /api/v1/faculties/:id
func (h *Handler) GetByID(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID fakultas tidak valid (harus UUID)"), reqID)
		return
	}

	result, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Data fakultas berhasil diambil", result, nil)
}

// List handles GET /api/v1/faculties
func (h *Handler) List(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	p, err := pagination.Parse(c)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	filter := FacultyFilter{
		Search: strings.TrimSpace(c.Query("search")),
	}

	if activeStr := strings.TrimSpace(c.Query("is_active")); activeStr != "" {
		if activeBool, err := strconv.ParseBool(activeStr); err == nil {
			filter.IsActive = &activeBool
		} else {
			response.RenderFromError(c, appErrors.ErrBadRequest("Parameter 'is_active' harus bernilai boolean (true/false)"), reqID)
			return
		}
	}

	result, meta, err := h.service.List(c.Request.Context(), filter, p)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Daftar fakultas berhasil diambil", result, meta)
}

// Update handles PATCH /api/v1/faculties/:id
func (h *Handler) Update(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID fakultas tidak valid (harus UUID)"), reqID)
		return
	}

	var req UpdateFacultyRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	result, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Data fakultas berhasil diperbarui", result, nil)
}

// UpdateStatus handles PATCH /api/v1/faculties/:id/status
func (h *Handler) UpdateStatus(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID fakultas tidak valid (harus UUID)"), reqID)
		return
	}

	var req UpdateStatusRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	result, err := h.service.UpdateStatus(c.Request.Context(), id, *req.IsActive)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Status fakultas berhasil diperbarui", result, nil)
}
