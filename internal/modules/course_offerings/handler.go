package course_offerings

import (
	"educore/internal/middleware"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"
	"educore/internal/shared/response"
	"educore/internal/shared/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateCourseOfferingRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	res, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	response.Created(c, "Penawaran mata kuliah berhasil dibuat", res, nil)
}

func (h *Handler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID penawaran mata kuliah tidak valid"), middleware.GetRequestID(c))
		return
	}

	res, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	response.OK(c, "Penawaran mata kuliah berhasil ditemukan", res, nil)
}

func (h *Handler) List(c *gin.Context) {
	p, err := pagination.Parse(c)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	var filter CourseOfferingFilter
	filter.Search = c.Query("search")
	filter.Status = c.Query("status")

	if periodIDStr := c.Query("academic_period_id"); periodIDStr != "" {
		if pid, err := uuid.Parse(periodIDStr); err == nil {
			filter.AcademicPeriodID = &pid
		}
	}
	if prodiIDStr := c.Query("study_program_id"); prodiIDStr != "" {
		if spid, err := uuid.Parse(prodiIDStr); err == nil {
			filter.StudyProgramID = &spid
		}
	}
	if courseIDStr := c.Query("course_id"); courseIDStr != "" {
		if cid, err := uuid.Parse(courseIDStr); err == nil {
			filter.CourseID = &cid
		}
	}
	if currIDStr := c.Query("curriculum_id"); currIDStr != "" {
		if currid, err := uuid.Parse(currIDStr); err == nil {
			filter.CurriculumID = &currid
		}
	}

	list, total, err := h.svc.List(c.Request.Context(), filter, p)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	meta := pagination.NewMeta(p.Page, p.Limit, total)
	response.OK(c, "Daftar penawaran mata kuliah berhasil diambil", list, meta)
}

func (h *Handler) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID penawaran mata kuliah tidak valid"), middleware.GetRequestID(c))
		return
	}

	var req UpdateCourseOfferingStatusRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	if err := h.svc.UpdateStatus(c.Request.Context(), id, req); err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	response.OK(c, "Status penawaran mata kuliah berhasil diperbarui", nil, nil)
}
