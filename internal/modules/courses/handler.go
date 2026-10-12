package courses

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

// Handler handles HTTP requests for course endpoints.
type Handler struct {
	service Service
	logger  *slog.Logger
}

// NewHandler instantiates a new course HTTP handler.
func NewHandler(service Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// Create handles POST /api/v1/courses
func (h *Handler) Create(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	var req CreateCourseRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	result, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.Created(c, "Mata kuliah berhasil dibuat", result, nil)
}

// GetByID handles GET /api/v1/courses/:id
func (h *Handler) GetByID(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID mata kuliah tidak valid (harus UUID)"), reqID)
		return
	}

	result, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Data mata kuliah berhasil diambil", result, nil)
}

// List handles GET /api/v1/courses
func (h *Handler) List(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	p, err := pagination.Parse(c)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	filter := ListCoursesFilter{
		Search: strings.TrimSpace(c.Query("search")),
	}

	if creditsStr := c.Query("credits"); creditsStr != "" {
		if credits, err := strconv.Atoi(creditsStr); err == nil {
			filter.Credits = &credits
		}
	}

	if activeStr := c.Query("is_active"); activeStr != "" {
		if active, err := strconv.ParseBool(activeStr); err == nil {
			filter.IsActive = &active
		}
	}

	list, meta, err := h.service.List(c.Request.Context(), filter, p)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Daftar mata kuliah berhasil diambil", list, meta)
}

// Update handles PATCH /api/v1/courses/:id
func (h *Handler) Update(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID mata kuliah tidak valid (harus UUID)"), reqID)
		return
	}

	var req UpdateCourseRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	result, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Data mata kuliah berhasil diperbarui", result, nil)
}

// UpdateStatus handles PATCH /api/v1/courses/:id/status
func (h *Handler) UpdateStatus(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID mata kuliah tidak valid (harus UUID)"), reqID)
		return
	}

	var req UpdateCourseStatusRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	result, err := h.service.UpdateStatus(c.Request.Context(), id, req)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Status mata kuliah berhasil diperbarui", result, nil)
}

// AddPrerequisite handles POST /api/v1/courses/:id/prerequisites
func (h *Handler) AddPrerequisite(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID mata kuliah tidak valid (harus UUID)"), reqID)
		return
	}

	var req AddPrerequisiteRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	if err := h.service.AddPrerequisite(c.Request.Context(), id, req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.Created(c, "Prasyarat mata kuliah berhasil ditambahkan", nil, nil)
}

// RemovePrerequisite handles DELETE /api/v1/courses/:id/prerequisites/:prerequisiteId
func (h *Handler) RemovePrerequisite(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	courseIDStr := c.Param("id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID mata kuliah tidak valid (harus UUID)"), reqID)
		return
	}

	prereqIDStr := c.Param("prerequisiteId")
	prereqID, err := uuid.Parse(prereqIDStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID prasyarat tidak valid (harus UUID)"), reqID)
		return
	}

	if err := h.service.RemovePrerequisite(c.Request.Context(), courseID, prereqID); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Prasyarat mata kuliah berhasil dihapus", nil, nil)
}
