package lecturers

import (
	"log/slog"
	"strings"

	"educore/internal/middleware"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"
	"educore/internal/shared/response"
	"educore/internal/shared/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for lecturer endpoints.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler instantiates a new lecturer HTTP handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// Create handles POST /api/v1/lecturers
func (h *Handler) Create(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	var req CreateLecturerRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	result, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.Created(c, "Profil dosen berhasil dibuat", result, nil)
}

// GetByID handles GET /api/v1/lecturers/:id
func (h *Handler) GetByID(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID dosen tidak valid (harus UUID)"), reqID)
		return
	}

	result, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Data dosen berhasil diambil", result, nil)
}

// List handles GET /api/v1/lecturers
func (h *Handler) List(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	p, err := pagination.Parse(c)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	filter := LecturerFilter{
		Search: strings.TrimSpace(c.Query("search")),
		Status: strings.ToUpper(strings.TrimSpace(c.Query("status"))),
	}

	if spIDStr := strings.TrimSpace(c.Query("study_program_id")); spIDStr != "" {
		if spID, err := uuid.Parse(spIDStr); err == nil {
			filter.StudyProgramID = &spID
		} else {
			response.RenderFromError(c, appErrors.ErrBadRequest("Parameter 'study_program_id' harus berupa UUID yang valid"), reqID)
			return
		}
	}

	result, meta, err := h.service.List(c.Request.Context(), filter, p)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Daftar dosen berhasil diambil", result, meta)
}

// Update handles PATCH /api/v1/lecturers/:id
func (h *Handler) Update(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID dosen tidak valid (harus UUID)"), reqID)
		return
	}

	var req UpdateLecturerRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	result, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Data dosen berhasil diperbarui", result, nil)
}

// UpdateStatus handles PATCH /api/v1/lecturers/:id/status
func (h *Handler) UpdateStatus(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID dosen tidak valid (harus UUID)"), reqID)
		return
	}

	var req UpdateStatusRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	result, err := h.service.UpdateStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Status dosen berhasil diperbarui", result, nil)
}
