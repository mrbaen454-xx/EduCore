package classes

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

func (h *Handler) CreateClass(c *gin.Context) {
	var req CreateClassRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	res, err := h.svc.CreateClass(c.Request.Context(), req)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	response.Created(c, "Kelas perkuliahan berhasil dibuat", res, nil)
}

func (h *Handler) GetClassByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID kelas tidak valid"), middleware.GetRequestID(c))
		return
	}

	res, err := h.svc.GetClassByID(c.Request.Context(), id)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	response.OK(c, "Kelas perkuliahan berhasil ditemukan", res, nil)
}

func (h *Handler) ListClasses(c *gin.Context) {
	p, err := pagination.Parse(c)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	var filter ClassFilter
	filter.Search = c.Query("search")
	filter.Status = c.Query("status")

	if periodIDStr := c.Query("academic_period_id"); periodIDStr != "" {
		if pid, err := uuid.Parse(periodIDStr); err == nil {
			filter.AcademicPeriodID = &pid
		}
	}
	if offeringIDStr := c.Query("course_offering_id"); offeringIDStr != "" {
		if oid, err := uuid.Parse(offeringIDStr); err == nil {
			filter.CourseOfferingID = &oid
		}
	}
	if prodiIDStr := c.Query("study_program_id"); prodiIDStr != "" {
		if spid, err := uuid.Parse(prodiIDStr); err == nil {
			filter.StudyProgramID = &spid
		}
	}

	list, total, err := h.svc.ListClasses(c.Request.Context(), filter, p)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	meta := pagination.NewMeta(p.Page, p.Limit, total)
	response.OK(c, "Daftar kelas berhasil diambil", list, meta)
}

func (h *Handler) UpdateClass(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID kelas tidak valid"), middleware.GetRequestID(c))
		return
	}

	var req UpdateClassRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	res, err := h.svc.UpdateClass(c.Request.Context(), id, req)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	response.OK(c, "Kelas perkuliahan berhasil diperbarui", res, nil)
}

func (h *Handler) UpdateClassStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID kelas tidak valid"), middleware.GetRequestID(c))
		return
	}

	var req UpdateClassStatusRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	if err := h.svc.UpdateClassStatus(c.Request.Context(), id, req); err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	response.OK(c, "Status kelas perkuliahan berhasil diperbarui", nil, nil)
}

// Lecturers

func (h *Handler) AssignLecturer(c *gin.Context) {
	idStr := c.Param("id")
	classID, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID kelas tidak valid"), middleware.GetRequestID(c))
		return
	}

	var req AssignLecturerRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	res, err := h.svc.AssignLecturer(c.Request.Context(), classID, req)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	response.Created(c, "Dosen berhasil ditugaskan ke kelas", res, nil)
}

func (h *Handler) GetClassLecturers(c *gin.Context) {
	idStr := c.Param("id")
	classID, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID kelas tidak valid"), middleware.GetRequestID(c))
		return
	}

	res, err := h.svc.GetClassByID(c.Request.Context(), classID)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	response.OK(c, "Daftar dosen pengajar kelas berhasil diambil", res.Lecturers, nil)
}

func (h *Handler) RemoveLecturer(c *gin.Context) {
	classIDStr := c.Param("id")
	classID, err := uuid.Parse(classIDStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID kelas tidak valid"), middleware.GetRequestID(c))
		return
	}

	lecturerIDStr := c.Param("lecturerId")
	lecturerID, err := uuid.Parse(lecturerIDStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID dosen tidak valid"), middleware.GetRequestID(c))
		return
	}

	res, err := h.svc.RemoveLecturer(c.Request.Context(), classID, lecturerID)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	response.OK(c, "Penugasan dosen berhasil dihapus dari kelas", res, nil)
}

// Schedules

func (h *Handler) CreateSchedule(c *gin.Context) {
	idStr := c.Param("id")
	classID, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID kelas tidak valid"), middleware.GetRequestID(c))
		return
	}

	var req CreateScheduleRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	res, err := h.svc.CreateSchedule(c.Request.Context(), classID, req)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	response.Created(c, "Jadwal perkuliahan berhasil dibuat", res, nil)
}

func (h *Handler) GetClassSchedules(c *gin.Context) {
	idStr := c.Param("id")
	classID, err := uuid.Parse(idStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID kelas tidak valid"), middleware.GetRequestID(c))
		return
	}

	res, err := h.svc.GetClassByID(c.Request.Context(), classID)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	response.OK(c, "Daftar jadwal perkuliahan berhasil diambil", res.Schedules, nil)
}

func (h *Handler) UpdateSchedule(c *gin.Context) {
	classIDStr := c.Param("id")
	classID, err := uuid.Parse(classIDStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID kelas tidak valid"), middleware.GetRequestID(c))
		return
	}

	scheduleIDStr := c.Param("scheduleId")
	scheduleID, err := uuid.Parse(scheduleIDStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID jadwal tidak valid"), middleware.GetRequestID(c))
		return
	}

	var req UpdateScheduleRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	res, err := h.svc.UpdateSchedule(c.Request.Context(), classID, scheduleID, req)
	if err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	response.OK(c, "Jadwal perkuliahan berhasil diperbarui", res, nil)
}

func (h *Handler) DeleteSchedule(c *gin.Context) {
	classIDStr := c.Param("id")
	classID, err := uuid.Parse(classIDStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID kelas tidak valid"), middleware.GetRequestID(c))
		return
	}

	scheduleIDStr := c.Param("scheduleId")
	scheduleID, err := uuid.Parse(scheduleIDStr)
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID jadwal tidak valid"), middleware.GetRequestID(c))
		return
	}

	if err := h.svc.DeleteSchedule(c.Request.Context(), classID, scheduleID); err != nil {
		response.RenderFromError(c, err, middleware.GetRequestID(c))
		return
	}

	response.OK(c, "Jadwal perkuliahan berhasil dihapus", nil, nil)
}
