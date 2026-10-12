package students

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

// Handler handles HTTP requests for student endpoints.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler instantiates a new student HTTP handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// Create handles POST /api/v1/students
func (h *Handler) Create(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	var req CreateStudentRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	result, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.Created(c, "Profil mahasiswa berhasil dibuat", result, nil)
}

// GetByID handles GET /api/v1/students/:id
func (h *Handler) GetByID(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID mahasiswa tidak valid (harus UUID)"), reqID)
		return
	}

	result, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Data mahasiswa berhasil diambil", result, nil)
}

// List handles GET /api/v1/students
func (h *Handler) List(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	p, err := pagination.Parse(c)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	filter := StudentFilter{
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

	if yearStr := strings.TrimSpace(c.Query("admission_year")); yearStr != "" {
		if year, err := strconv.Atoi(yearStr); err == nil {
			filter.AdmissionYear = &year
		} else {
			response.RenderFromError(c, appErrors.ErrBadRequest("Parameter 'admission_year' harus berupa angka tahun yang valid"), reqID)
			return
		}
	}

	result, meta, err := h.service.List(c.Request.Context(), filter, p)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Daftar mahasiswa berhasil diambil", result, meta)
}

// Update handles PATCH /api/v1/students/:id
func (h *Handler) Update(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID mahasiswa tidak valid (harus UUID)"), reqID)
		return
	}

	var req UpdateStudentRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	result, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Data mahasiswa berhasil diperbarui", result, nil)
}

// UpdateStatus handles PATCH /api/v1/students/:id/status
func (h *Handler) UpdateStatus(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID mahasiswa tidak valid (harus UUID)"), reqID)
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

	response.OK(c, "Status mahasiswa berhasil diperbarui", result, nil)
}
