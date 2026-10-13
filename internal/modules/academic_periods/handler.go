package academic_periods

import (
	"strconv"

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
	var req CreateAcademicPeriodRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	res, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	response.Created(c, "Periode akademik berhasil dibuat", res, nil)
}

func (h *Handler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID periode akademik tidak valid"), middleware.GetRequestID(c))
		return
	}

	res, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	response.OK(c, "Periode akademik berhasil ditemukan", res, nil)
}

func (h *Handler) List(c *gin.Context) {
	p, err := pagination.Parse(c)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	var filter AcademicPeriodFilter
	filter.Search = c.Query("search")
	filter.SemesterType = c.Query("semester_type")

	if activeStr := c.Query("is_active"); activeStr != "" {
		if activeBool, err := strconv.ParseBool(activeStr); err == nil {
			filter.IsActive = &activeBool
		}
	}

	list, total, err := h.svc.List(c.Request.Context(), filter, p)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	meta := pagination.NewMeta(p.Page, p.Limit, total)
	response.OK(c, "Daftar periode akademik berhasil diambil", list, meta)
}

func (h *Handler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID periode akademik tidak valid"), middleware.GetRequestID(c))
		return
	}

	var req UpdateAcademicPeriodRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	res, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	response.OK(c, "Periode akademik berhasil diperbarui", res, nil)
}
