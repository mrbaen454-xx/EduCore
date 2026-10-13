package study_plans

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
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Student Endpoints

// GetStudentAvailableClasses retrieves open class offerings available for student enrollment.
func (h *Handler) GetStudentAvailableClasses(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	p, err := pagination.Parse(c)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	var periodID *uuid.UUID
	if periodStr := c.Query("period_id"); periodStr != "" {
		parsed, err := uuid.Parse(periodStr)
		if err != nil {
			response.RenderFromError(c, appErrors.ErrBadRequest("Format period_id tidak valid"), reqID)
			return
		}
		periodID = &parsed
	} else {
		// Default to currently active academic period if not specified
		activeID, err := h.service.GetActiveAcademicPeriodID(c.Request.Context())
		if err == nil {
			periodID = &activeID
		}
	}

	var spID *uuid.UUID
	if spStr := c.Query("study_program_id"); spStr != "" {
		parsed, err := uuid.Parse(spStr)
		if err != nil {
			response.RenderFromError(c, appErrors.ErrBadRequest("Format study_program_id tidak valid"), reqID)
			return
		}
		spID = &parsed
	}

	filter := AvailableClassFilter{
		AcademicPeriodID: periodID,
		StudyProgramID:   spID,
		Search:           c.Query("search"),
	}

	list, total, err := h.service.ListAvailableClasses(c.Request.Context(), filter, p)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	dtos := make([]AvailableClassResponse, len(list))
	for i, ac := range list {
		dtos[i] = toAvailableClassResponse(&ac)
	}

	meta := pagination.NewMeta(p.Page, p.Limit, total)
	response.OK(c, "Daftar penawaran kelas berhasil diambil", dtos, meta)
}

// GetMyStudyPlans lists all study plans belonging to the authenticated student.
func (h *Handler) GetMyStudyPlans(c *gin.Context) {
	reqID := middleware.GetRequestID(c)
	principal, _ := middleware.GetPrincipal(c)

	p, err := pagination.Parse(c)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	var periodID *uuid.UUID
	if periodStr := c.Query("period_id"); periodStr != "" {
		parsed, err := uuid.Parse(periodStr)
		if err != nil {
			response.RenderFromError(c, appErrors.ErrBadRequest("Format period_id tidak valid"), reqID)
			return
		}
		periodID = &parsed
	}

	filter := StudyPlanFilter{
		AcademicPeriodID: periodID,
		Status:           c.Query("status"),
	}

	list, total, err := h.service.ListMyStudyPlans(c.Request.Context(), principal, filter, p)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	dtos := make([]StudyPlanResponse, len(list))
	for i, sp := range list {
		dtos[i] = toStudyPlanResponse(&sp)
	}

	meta := pagination.NewMeta(p.Page, p.Limit, total)
	response.OK(c, "Daftar KRS mahasiswa berhasil diambil", dtos, meta)
}

// CreateMyStudyPlan creates or retrieves the authenticated student's study plan for a period.
func (h *Handler) CreateMyStudyPlan(c *gin.Context) {
	reqID := middleware.GetRequestID(c)
	principal, _ := middleware.GetPrincipal(c)

	var req CreateStudyPlanRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	sp, created, err := h.service.GetOrCreateMyStudyPlan(c.Request.Context(), principal, req.AcademicPeriodID)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	dto := toStudyPlanDetailResponse(sp)
	if created {
		response.Created(c, "KRS berhasil dibuat", dto, nil)
	} else {
		response.OK(c, "KRS untuk periode ini sudah ada dan berhasil diambil", dto, nil)
	}
}

// GetMyStudyPlanDetail retrieves full details of a specific study plan owned by the student.
func (h *Handler) GetMyStudyPlanDetail(c *gin.Context) {
	reqID := middleware.GetRequestID(c)
	principal, _ := middleware.GetPrincipal(c)

	spID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID KRS tidak valid"), reqID)
		return
	}

	sp, err := h.service.GetMyStudyPlanDetail(c.Request.Context(), principal, spID)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Detail KRS berhasil diambil", toStudyPlanDetailResponse(sp), nil)
}

// EnrollClassInMyStudyPlan enrolls a class into the student's study plan.
func (h *Handler) EnrollClassInMyStudyPlan(c *gin.Context) {
	reqID := middleware.GetRequestID(c)
	principal, _ := middleware.GetPrincipal(c)

	spID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID KRS tidak valid"), reqID)
		return
	}

	var req EnrollClassRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	enrolled, err := h.service.EnrollClassInMyStudyPlan(c.Request.Context(), principal, spID, req.ClassID)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.Created(c, "Kelas berhasil ditambahkan ke KRS", toEnrollmentResponse(enrolled), nil)
}

// DropEnrollmentFromMyStudyPlan drops a class enrollment from the student's study plan.
func (h *Handler) DropEnrollmentFromMyStudyPlan(c *gin.Context) {
	reqID := middleware.GetRequestID(c)
	principal, _ := middleware.GetPrincipal(c)

	spID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID KRS tidak valid"), reqID)
		return
	}

	enrollmentID, err := uuid.Parse(c.Param("enrollmentId"))
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID enrollment tidak valid"), reqID)
		return
	}

	if err := h.service.DropEnrollmentFromMyStudyPlan(c.Request.Context(), principal, spID, enrollmentID); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Registrasi kelas berhasil dibatalkan", gin.H{
		"study_plan_id": spID,
		"enrollment_id": enrollmentID,
		"status":        "DROPPED",
	}, nil)
}

// Administrative Endpoints

// ListStudyPlans lists all study plans in the system for admin/staff.
func (h *Handler) ListStudyPlans(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	p, err := pagination.Parse(c)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	var studentID *uuid.UUID
	if sStr := c.Query("student_id"); sStr != "" {
		parsed, err := uuid.Parse(sStr)
		if err != nil {
			response.RenderFromError(c, appErrors.ErrBadRequest("Format student_id tidak valid"), reqID)
			return
		}
		studentID = &parsed
	}

	var periodID *uuid.UUID
	if pStr := c.Query("period_id"); pStr != "" {
		parsed, err := uuid.Parse(pStr)
		if err != nil {
			response.RenderFromError(c, appErrors.ErrBadRequest("Format period_id tidak valid"), reqID)
			return
		}
		periodID = &parsed
	}

	filter := StudyPlanFilter{
		StudentID:        studentID,
		AcademicPeriodID: periodID,
		Status:           c.Query("status"),
	}

	list, total, err := h.service.ListStudyPlansForAdmin(c.Request.Context(), filter, p)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	dtos := make([]StudyPlanResponse, len(list))
	for i, sp := range list {
		dtos[i] = toStudyPlanResponse(&sp)
	}

	meta := pagination.NewMeta(p.Page, p.Limit, total)
	response.OK(c, "Daftar KRS berhasil diambil", dtos, meta)
}

// GetStudyPlanDetail retrieves full details of any study plan for admin/staff.
func (h *Handler) GetStudyPlanDetail(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	spID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID KRS tidak valid"), reqID)
		return
	}

	sp, err := h.service.GetStudyPlanDetailForAdmin(c.Request.Context(), spID)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Detail KRS berhasil diambil", toStudyPlanDetailResponse(sp), nil)
}

// CreateStudyPlan creates a study plan on behalf of a student.
func (h *Handler) CreateStudyPlan(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	var req CreateStudyPlanRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	sp, created, err := h.service.CreateStudyPlanForAdmin(c.Request.Context(), req)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	dto := toStudyPlanDetailResponse(sp)
	if created {
		response.Created(c, "KRS berhasil dibuat", dto, nil)
	} else {
		response.OK(c, "KRS mahasiswa untuk periode ini sudah ada dan berhasil diambil", dto, nil)
	}
}

// EnrollClass enrolls a class into a study plan on behalf of a student.
func (h *Handler) EnrollClass(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	spID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID KRS tidak valid"), reqID)
		return
	}

	var req EnrollClassRequest
	if err := validation.BindJSON(c, &req); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	enrolled, err := h.service.EnrollClassForAdmin(c.Request.Context(), spID, req.ClassID)
	if err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.Created(c, "Kelas berhasil didaftarkan ke KRS", toEnrollmentResponse(enrolled), nil)
}

// DropEnrollment drops a class enrollment from a study plan on behalf of a student.
func (h *Handler) DropEnrollment(c *gin.Context) {
	reqID := middleware.GetRequestID(c)

	spID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID KRS tidak valid"), reqID)
		return
	}

	enrollmentID, err := uuid.Parse(c.Param("enrollmentId"))
	if err != nil {
		response.RenderFromError(c, appErrors.ErrBadRequest("ID enrollment tidak valid"), reqID)
		return
	}

	if err := h.service.DropEnrollmentForAdmin(c.Request.Context(), spID, enrollmentID); err != nil {
		response.RenderFromError(c, err, reqID)
		return
	}

	response.OK(c, "Registrasi kelas berhasil dibatalkan", gin.H{
		"study_plan_id": spID,
		"enrollment_id": enrollmentID,
		"status":        "DROPPED",
	}, nil)
}

// Helper converters

func toStudyPlanResponse(d *StudyPlanDetail) StudyPlanResponse {
	return StudyPlanResponse{
		ID:                 d.ID,
		StudentID:          d.StudentID,
		StudentNIM:         d.StudentNIM,
		StudentName:        d.StudentName,
		StudyProgramID:     d.StudyProgramID,
		StudyProgramName:   d.StudyProgramName,
		AcademicPeriodID:   d.AcademicPeriodID,
		AcademicPeriodCode: d.AcademicPeriodCode,
		AcademicPeriodName: d.AcademicPeriodName,
		TotalCredits:       d.TotalCredits,
		Status:             d.Status,
		CreatedAt:          d.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:          d.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toEnrollmentResponse(e *EnrollmentDetail) EnrollmentResponse {
	res := EnrollmentResponse{
		ID:            e.ID,
		StudyPlanID:   e.StudyPlanID,
		StudentID:     e.StudentID,
		ClassID:       e.ClassID,
		ClassCode:     e.ClassCode,
		ClassName:     e.ClassName,
		CourseID:      e.CourseID,
		CourseCode:    e.CourseCode,
		CourseName:    e.CourseName,
		CourseCredits: e.CourseCredits,
		Status:        e.Status,
		EnrolledAt:    e.EnrolledAt.Format("2006-01-02T15:04:05Z07:00"),
		Lecturers:     e.Lecturers,
		Schedules:     e.Schedules,
	}
	if e.DroppedAt != nil {
		formatted := e.DroppedAt.Format("2006-01-02T15:04:05Z07:00")
		res.DroppedAt = &formatted
	}
	return res
}

func toStudyPlanDetailResponse(d *StudyPlanDetail) StudyPlanDetailResponse {
	enrollments := make([]EnrollmentResponse, len(d.Enrollments))
	for i, e := range d.Enrollments {
		enrollments[i] = toEnrollmentResponse(&e)
	}

	return StudyPlanDetailResponse{
		StudyPlanResponse: toStudyPlanResponse(d),
		Enrollments:       enrollments,
	}
}

func toAvailableClassResponse(ac *AvailableClass) AvailableClassResponse {
	return AvailableClassResponse{
		ClassID:            ac.ClassID,
		ClassCode:          ac.ClassCode,
		ClassName:          ac.ClassName,
		Capacity:           ac.Capacity,
		EnrolledCount:      ac.EnrolledCount,
		AvailableSeats:     ac.AvailableSeats,
		ClassStatus:        ac.ClassStatus,
		CourseOfferingID:   ac.CourseOfferingID,
		OfferingStatus:     ac.OfferingStatus,
		CourseID:           ac.CourseID,
		CourseCode:         ac.CourseCode,
		CourseName:         ac.CourseName,
		CourseCredits:      ac.CourseCredits,
		AcademicPeriodID:   ac.AcademicPeriodID,
		AcademicPeriodCode: ac.AcademicPeriodCode,
		StudyProgramID:     ac.StudyProgramID,
		StudyProgramName:   ac.StudyProgramName,
		Lecturers:          ac.Lecturers,
		Schedules:          ac.Schedules,
	}
}
