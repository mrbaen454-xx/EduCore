package attendance

import (
	"errors"

	"educore/internal/middleware"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"
	"educore/internal/shared/response"
	"educore/internal/shared/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for class sessions and student attendance.
type Handler struct {
	service Service
}

// NewHandler constructs a new Handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func parseUUIDParam(c *gin.Context, keys ...string) (uuid.UUID, error) {
	for _, k := range keys {
		val := c.Param(k)
		if val != "" {
			return uuid.Parse(val)
		}
	}
	return uuid.Nil, errors.New("parameter missing")
}

// CreateSession handles POST /api/v1/classes/:id/sessions
func (h *Handler) CreateSession(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	classID, err := parseUUIDParam(c, "id", "classId")
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID kelas tidak valid"), reqID)
		return
	}

	var req CreateSessionRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	principal, _ := middleware.GetPrincipal(c)
	resp, err := h.service.CreateSession(c.Request.Context(), classID, req, principal)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.Created(c, "Sesi perkuliahan berhasil dibuat", resp, nil)
}

// ListSessionsByClass handles GET /api/v1/classes/:id/sessions
func (h *Handler) ListSessionsByClass(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	classID, err := parseUUIDParam(c, "id", "classId")
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID kelas tidak valid"), reqID)
		return
	}

	p, err := pagination.Parse(c)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	principal, _ := middleware.GetPrincipal(c)
	sessions, meta, err := h.service.ListSessionsByClass(c.Request.Context(), classID, p, principal)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Daftar sesi perkuliahan berhasil diambil", sessions, meta)
}

// GetSessionDetail handles GET /api/v1/sessions/:id
func (h *Handler) GetSessionDetail(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	sessionID, err := parseUUIDParam(c, "id", "sessionId")
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID sesi tidak valid"), reqID)
		return
	}

	principal, _ := middleware.GetPrincipal(c)
	session, err := h.service.GetSessionDetail(c.Request.Context(), sessionID, principal)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Detail sesi perkuliahan berhasil diambil", session, nil)
}

// UpdateSession handles PATCH /api/v1/sessions/:id
func (h *Handler) UpdateSession(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	sessionID, err := parseUUIDParam(c, "id", "sessionId")
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID sesi tidak valid"), reqID)
		return
	}

	var req UpdateSessionRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	principal, _ := middleware.GetPrincipal(c)
	session, err := h.service.UpdateSession(c.Request.Context(), sessionID, req, principal)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Sesi perkuliahan berhasil diperbarui", session, nil)
}

// CloseSession handles POST /api/v1/sessions/:id/close
func (h *Handler) CloseSession(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	sessionID, err := parseUUIDParam(c, "id", "sessionId")
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID sesi tidak valid"), reqID)
		return
	}

	principal, _ := middleware.GetPrincipal(c)
	if err := h.service.CloseSession(c.Request.Context(), sessionID, principal); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Sesi perkuliahan berhasil ditutup", nil, nil)
}

// GetSessionRoster handles GET /api/v1/sessions/:id/students
func (h *Handler) GetSessionRoster(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	sessionID, err := parseUUIDParam(c, "id", "sessionId")
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID sesi tidak valid"), reqID)
		return
	}

	principal, _ := middleware.GetPrincipal(c)
	roster, err := h.service.GetSessionRoster(c.Request.Context(), sessionID, principal)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Daftar peserta sesi perkuliahan berhasil diambil", roster, nil)
}

// GetSessionAttendance handles GET /api/v1/sessions/:id/attendance
func (h *Handler) GetSessionAttendance(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	sessionID, err := parseUUIDParam(c, "id", "sessionId")
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID sesi tidak valid"), reqID)
		return
	}

	principal, _ := middleware.GetPrincipal(c)
	attendances, summary, err := h.service.GetSessionAttendance(c.Request.Context(), sessionID, principal)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Data presensi sesi perkuliahan berhasil diambil", gin.H{
		"session_id":  sessionID,
		"attendances": attendances,
		"summary":     summary,
	}, nil)
}

// RecordBatchAttendance handles POST /api/v1/sessions/:id/attendance
func (h *Handler) RecordBatchAttendance(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	sessionID, err := parseUUIDParam(c, "id", "sessionId")
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID sesi tidak valid"), reqID)
		return
	}

	var req RecordBatchAttendanceRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	principal, _ := middleware.GetPrincipal(c)
	if err := h.service.RecordBatchAttendance(c.Request.Context(), sessionID, req, principal); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Presensi mahasiswa berhasil dicatat", nil, nil)
}

// UpdateAttendance handles PATCH /api/v1/sessions/:id/attendance/:attendanceId
func (h *Handler) UpdateAttendance(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	sessionID, err := parseUUIDParam(c, "id", "sessionId")
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID sesi tidak valid"), reqID)
		return
	}

	attendanceID, err := parseUUIDParam(c, "attendanceId", "attendance_id")
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID presensi tidak valid"), reqID)
		return
	}

	var req UpdateAttendanceRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	principal, _ := middleware.GetPrincipal(c)
	if err := h.service.UpdateAttendance(c.Request.Context(), sessionID, attendanceID, req, principal); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Presensi mahasiswa berhasil diperbarui", nil, nil)
}

// GetClassAttendanceSummary handles GET /api/v1/classes/:id/attendance/summary
func (h *Handler) GetClassAttendanceSummary(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	classID, err := parseUUIDParam(c, "id", "classId")
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID kelas tidak valid"), reqID)
		return
	}

	principal, _ := middleware.GetPrincipal(c)
	summary, err := h.service.GetClassAttendanceSummary(c.Request.Context(), classID, principal)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Rekapitulasi kehadiran kelas berhasil diambil", summary, nil)
}

// GetMyAttendanceHistory handles GET /api/v1/student/attendance
func (h *Handler) GetMyAttendanceHistory(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	p, err := pagination.Parse(c)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	principal, _ := middleware.GetPrincipal(c)
	history, meta, summary, err := h.service.GetStudentAttendanceHistory(c.Request.Context(), nil, p, principal)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Riwayat presensi mahasiswa berhasil diambil", gin.H{
		"attendances": history,
		"summary":     summary,
	}, meta)
}

// GetMyClassAttendance handles GET /api/v1/student/attendance/classes/:id
func (h *Handler) GetMyClassAttendance(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	classID, err := parseUUIDParam(c, "id", "classId")
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("Format ID kelas tidak valid"), reqID)
		return
	}

	p, err := pagination.Parse(c)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	principal, _ := middleware.GetPrincipal(c)
	history, meta, summary, err := h.service.GetStudentAttendanceHistory(c.Request.Context(), &classID, p, principal)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Riwayat presensi kelas mahasiswa berhasil diambil", gin.H{
		"class_id":    classID,
		"attendances": history,
		"summary":     summary,
	}, meta)
}
