package curricula

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

// Handler handles HTTP requests for curriculum and curriculum course endpoints.
type Handler struct {
	service Service
	logger  *slog.Logger
}

// NewHandler instantiates a new curriculum HTTP handler.
func NewHandler(service Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// Create handles POST /api/v1/curricula
func (h *Handler) Create(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	var req CreateCurriculumRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	result, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.Created(c, "Kurikulum berhasil dibuat", result, nil)
}

// GetByID handles GET /api/v1/curricula/:id
func (h *Handler) GetByID(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID kurikulum tidak valid (harus UUID)"), reqID)
		return
	}

	result, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Data kurikulum berhasil diambil", result, nil)
}

// List handles GET /api/v1/curricula
func (h *Handler) List(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	p, err := pagination.Parse(c)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	filter := ListCurriculaFilter{
		Search: strings.TrimSpace(c.Query("search")),
		Status: strings.TrimSpace(c.Query("status")),
	}

	if spStr := c.Query("study_program_id"); spStr != "" {
		if spID, err := uuid.Parse(spStr); err == nil {
			filter.StudyProgramID = &spID
		}
	}

	if yearStr := c.Query("year"); yearStr != "" {
		if year, err := strconv.Atoi(yearStr); err == nil {
			filter.Year = &year
		}
	}

	list, meta, err := h.service.List(c.Request.Context(), filter, p)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Daftar kurikulum berhasil diambil", list, meta)
}

// Update handles PATCH /api/v1/curricula/:id
func (h *Handler) Update(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID kurikulum tidak valid (harus UUID)"), reqID)
		return
	}

	var req UpdateCurriculumRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	result, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Data kurikulum berhasil diperbarui", result, nil)
}

// UpdateStatus handles PATCH /api/v1/curricula/:id/status
func (h *Handler) UpdateStatus(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID kurikulum tidak valid (harus UUID)"), reqID)
		return
	}

	var req UpdateCurriculumStatusRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	result, err := h.service.UpdateStatus(c.Request.Context(), id, req)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Status kurikulum berhasil diperbarui", result, nil)
}

// AddCourse handles POST /api/v1/curricula/:id/courses
func (h *Handler) AddCourse(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	idStr := c.Param("id")
	curriculumID, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID kurikulum tidak valid (harus UUID)"), reqID)
		return
	}

	var req AddCurriculumCourseRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	result, err := h.service.AddCourse(c.Request.Context(), curriculumID, req)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.Created(c, "Mata kuliah berhasil ditambahkan ke kurikulum", result, nil)
}

// GetCourse handles GET /api/v1/curricula/:id/courses/:courseId
func (h *Handler) GetCourse(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	curriculumID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID kurikulum tidak valid (harus UUID)"), reqID)
		return
	}

	courseID, err := uuid.Parse(c.Param("courseId"))
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID mata kuliah tidak valid (harus UUID)"), reqID)
		return
	}

	result, err := h.service.GetCourse(c.Request.Context(), curriculumID, courseID)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Detail mata kuliah dalam kurikulum berhasil diambil", result, nil)
}

// ListCourses handles GET /api/v1/curricula/:id/courses
func (h *Handler) ListCourses(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	curriculumID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID kurikulum tidak valid (harus UUID)"), reqID)
		return
	}

	p, err := pagination.Parse(c)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	filter := ListCurriculumCoursesFilter{
		Category: strings.TrimSpace(c.Query("category")),
	}

	if semStr := c.Query("semester"); semStr != "" {
		if sem, err := strconv.Atoi(semStr); err == nil {
			filter.Semester = &sem
		}
	}

	list, meta, err := h.service.ListCourses(c.Request.Context(), curriculumID, filter, p)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Daftar mata kuliah dalam kurikulum berhasil diambil", list, meta)
}

// UpdateCourse handles PATCH /api/v1/curricula/:id/courses/:courseId
func (h *Handler) UpdateCourse(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	curriculumID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID kurikulum tidak valid (harus UUID)"), reqID)
		return
	}

	courseID, err := uuid.Parse(c.Param("courseId"))
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID mata kuliah tidak valid (harus UUID)"), reqID)
		return
	}

	var req UpdateCurriculumCourseRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	result, err := h.service.UpdateCourse(c.Request.Context(), curriculumID, courseID, req)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Mata kuliah dalam kurikulum berhasil diperbarui", result, nil)
}

// RemoveCourse handles DELETE /api/v1/curricula/:id/courses/:courseId
func (h *Handler) RemoveCourse(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	curriculumID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID kurikulum tidak valid (harus UUID)"), reqID)
		return
	}

	courseID, err := uuid.Parse(c.Param("courseId"))
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID mata kuliah tidak valid (harus UUID)"), reqID)
		return
	}

	if err := h.service.RemoveCourse(c.Request.Context(), curriculumID, courseID); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Mata kuliah berhasil dihapus dari kurikulum", nil, nil)
}
