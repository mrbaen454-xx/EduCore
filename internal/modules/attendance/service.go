package attendance

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"educore/internal/modules/auth"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
)

const (
	MaxBatchAttendanceItems = 200
)

// Service defines business logic operations for lecture sessions and student attendance.
type Service interface {
	// Session management
	CreateSession(ctx context.Context, classID uuid.UUID, req CreateSessionRequest, principal *auth.Principal) (*SessionResponse, error)
	GetSessionDetail(ctx context.Context, sessionID uuid.UUID, principal *auth.Principal) (*SessionResponse, error)
	ListSessionsByClass(ctx context.Context, classID uuid.UUID, p pagination.Params, principal *auth.Principal) ([]SessionResponse, pagination.Meta, error)
	UpdateSession(ctx context.Context, sessionID uuid.UUID, req UpdateSessionRequest, principal *auth.Principal) (*SessionResponse, error)
	CloseSession(ctx context.Context, sessionID uuid.UUID, principal *auth.Principal) error

	// Session roster & attendance
	GetSessionRoster(ctx context.Context, sessionID uuid.UUID, principal *auth.Principal) (*SessionRosterResponse, error)
	GetSessionAttendance(ctx context.Context, sessionID uuid.UUID, principal *auth.Principal) ([]AttendanceDetailResponse, AttendanceSummary, error)
	RecordBatchAttendance(ctx context.Context, sessionID uuid.UUID, req RecordBatchAttendanceRequest, principal *auth.Principal) error
	UpdateAttendance(ctx context.Context, sessionID, attendanceID uuid.UUID, req UpdateAttendanceRequest, principal *auth.Principal) error

	// Student history (self-service)
	GetStudentAttendanceHistory(ctx context.Context, classID *uuid.UUID, p pagination.Params, principal *auth.Principal) ([]StudentAttendanceHistory, pagination.Meta, AttendanceSummary, error)

	// Class cumulative report
	GetClassAttendanceSummary(ctx context.Context, classID uuid.UUID, principal *auth.Principal) (*ClassAttendanceSummary, error)
}

type service struct {
	repo   Repository
	logger *slog.Logger
}

// NewService constructs a new attendance service instance.
func NewService(repo Repository, logger *slog.Logger) Service {
	return &service{
		repo:   repo,
		logger: logger,
	}
}

// authorizeClassManagement verifies that principal is admin, akademik, or the lecturer teaching this class.
func (s *service) authorizeClassManagement(ctx context.Context, classID uuid.UUID, principal *auth.Principal) error {
	if principal == nil {
		return appErrors.ErrUnauthorized("Autentikasi diperlukan")
	}

	if principal.HasAnyRole("admin", "akademik") {
		return nil
	}

	if principal.HasRole("dosen") {
		isTeaching, err := s.repo.CheckLecturerTeachingClass(ctx, classID, principal.UserID)
		if err != nil {
			return err
		}
		if !isTeaching {
			return ErrUnauthorizedLecturer
		}
		return nil
	}

	return appErrors.ErrForbidden("Anda tidak memiliki izin mengelola presensi kelas ini")
}

// authorizeReadClassSessions verifies principal has right to view sessions of the class (lecturer/staff or enrolled student).
func (s *service) authorizeReadClassSessions(ctx context.Context, classID uuid.UUID, principal *auth.Principal) error {
	if principal == nil {
		return appErrors.ErrUnauthorized("Autentikasi diperlukan")
	}

	if principal.HasAnyRole("admin", "akademik") {
		return nil
	}

	if principal.HasRole("dosen") {
		isTeaching, err := s.repo.CheckLecturerTeachingClass(ctx, classID, principal.UserID)
		if err != nil {
			return err
		}
		if !isTeaching {
			return ErrUnauthorizedLecturer
		}
		return nil
	}

	if principal.HasRole("mahasiswa") {
		student, err := s.repo.GetStudentByUserID(ctx, principal.UserID)
		if err != nil {
			return err
		}
		isEnrolled, err := s.repo.CheckStudentEnrolledInClass(ctx, classID, student.ID)
		if err != nil {
			return err
		}
		if !isEnrolled {
			return appErrors.ErrForbidden("Anda tidak terdaftar sebagai peserta pada kelas ini")
		}
		return nil
	}

	return appErrors.ErrForbidden("Akses ditolak")
}

func validationError(field, message string) *appErrors.AppError {
	return appErrors.ErrValidation([]appErrors.Detail{
		{Field: field, Message: message},
	})
}

// parseTime parses HH:MM or HH:MM:SS strings for comparison.
func parseTime(tStr string) (time.Time, error) {
	trimmed := strings.TrimSpace(tStr)
	if t, err := time.Parse("15:04:05", trimmed); err == nil {
		return t, nil
	}
	if t, err := time.Parse("15:04", trimmed); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("format waktu '%s' tidak valid, gunakan format 'HH:MM' atau 'HH:MM:SS'", tStr)
}

func (s *service) CreateSession(ctx context.Context, classID uuid.UUID, req CreateSessionRequest, principal *auth.Principal) (*SessionResponse, error) {
	if err := s.authorizeClassManagement(ctx, classID, principal); err != nil {
		return nil, err
	}

	parsedDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.Date))
	if err != nil {
		return nil, validationError("date", "Format tanggal sesi harus YYYY-MM-DD")
	}

	startTime, err := parseTime(req.StartTime)
	if err != nil {
		return nil, validationError("start_time", err.Error())
	}

	endTime, err := parseTime(req.EndTime)
	if err != nil {
		return nil, validationError("end_time", err.Error())
	}

	if !endTime.After(startTime) {
		return nil, validationError("end_time", "Waktu selesai sesi perkuliahan harus setelah waktu mulai")
	}

	topic := strings.TrimSpace(req.Topic)
	if topic == "" {
		return nil, validationError("topic", "Topik pertemuan perkuliahan tidak boleh kosong")
	}

	status := "OPEN"
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		st := strings.ToUpper(strings.TrimSpace(*req.Status))
		if st != "SCHEDULED" && st != "OPEN" {
			return nil, validationError("status", "Status awal sesi hanya boleh 'SCHEDULED' atau 'OPEN'")
		}
		status = st
	}

	var lecturerID *uuid.UUID
	if req.LecturerID != nil {
		lecturerID = req.LecturerID
	} else if principal.HasRole("dosen") {
		lec, err := s.repo.GetLecturerByUserID(ctx, principal.UserID)
		if err == nil && lec != nil {
			lecturerID = &lec.ID
		}
	}

	session := &ClassSession{
		ID:            uuid.New(),
		ClassID:       classID,
		SessionNumber: req.SessionNumber,
		Date:          parsedDate,
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
		Topic:         topic,
		Remarks:       req.Remarks,
		LecturerID:    lecturerID,
		Status:        status,
		CreatedBy:     &principal.UserID,
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	return s.GetSessionDetail(ctx, session.ID, principal)
}

func (s *service) GetSessionDetail(ctx context.Context, sessionID uuid.UUID, principal *auth.Principal) (*SessionResponse, error) {
	detail, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if err := s.authorizeReadClassSessions(ctx, detail.ClassID, principal); err != nil {
		return nil, err
	}

	resp := mapSessionDetailToResponse(detail)
	return &resp, nil
}

func (s *service) ListSessionsByClass(ctx context.Context, classID uuid.UUID, p pagination.Params, principal *auth.Principal) ([]SessionResponse, pagination.Meta, error) {
	if err := s.authorizeReadClassSessions(ctx, classID, principal); err != nil {
		return nil, pagination.Meta{}, err
	}

	details, total, err := s.repo.ListSessionsByClass(ctx, classID, p)
	if err != nil {
		return nil, pagination.Meta{}, err
	}

	responses := make([]SessionResponse, len(details))
	for i, d := range details {
		responses[i] = mapSessionDetailToResponse(&d)
	}

	meta := pagination.NewMeta(p.Page, p.Limit, total)
	return responses, meta, nil
}

func (s *service) UpdateSession(ctx context.Context, sessionID uuid.UUID, req UpdateSessionRequest, principal *auth.Principal) (*SessionResponse, error) {
	existing, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if err := s.authorizeClassManagement(ctx, existing.ClassID, principal); err != nil {
		return nil, err
	}

	if existing.Status == "CANCELLED" {
		return nil, ErrSessionCancelled
	}

	updated := existing.ClassSession

	if req.SessionNumber != nil && *req.SessionNumber > 0 {
		updated.SessionNumber = *req.SessionNumber
	}

	if req.Date != nil && strings.TrimSpace(*req.Date) != "" {
		parsedDate, err := time.Parse("2006-01-02", strings.TrimSpace(*req.Date))
		if err != nil {
			return nil, validationError("date", "Format tanggal sesi harus YYYY-MM-DD")
		}
		updated.Date = parsedDate
	}

	var startTime time.Time
	var endTime time.Time

	if req.StartTime != nil && strings.TrimSpace(*req.StartTime) != "" {
		updated.StartTime = strings.TrimSpace(*req.StartTime)
	}
	t, err := parseTime(updated.StartTime)
	if err != nil {
		return nil, validationError("start_time", err.Error())
	}
	startTime = t

	if req.EndTime != nil && strings.TrimSpace(*req.EndTime) != "" {
		updated.EndTime = strings.TrimSpace(*req.EndTime)
	}
	t, err = parseTime(updated.EndTime)
	if err != nil {
		return nil, validationError("end_time", err.Error())
	}
	endTime = t

	if !endTime.After(startTime) {
		return nil, validationError("end_time", "Waktu selesai sesi perkuliahan harus setelah waktu mulai")
	}

	if req.Topic != nil {
		trimmed := strings.TrimSpace(*req.Topic)
		if trimmed == "" {
			return nil, validationError("topic", "Topik pertemuan tidak boleh kosong")
		}
		updated.Topic = trimmed
	}

	if req.Remarks != nil {
		updated.Remarks = req.Remarks
	}

	if req.LecturerID != nil {
		updated.LecturerID = req.LecturerID
	}

	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		st := strings.ToUpper(strings.TrimSpace(*req.Status))
		switch st {
		case "SCHEDULED", "OPEN", "COMPLETED", "CANCELLED":
			updated.Status = st
		default:
			return nil, validationError("status", "Status sesi tidak valid. Pilihan: SCHEDULED, OPEN, COMPLETED, CANCELLED")
		}
	}

	if err := s.repo.UpdateSession(ctx, &updated); err != nil {
		return nil, err
	}

	return s.GetSessionDetail(ctx, sessionID, principal)
}

func (s *service) CloseSession(ctx context.Context, sessionID uuid.UUID, principal *auth.Principal) error {
	existing, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	if err := s.authorizeClassManagement(ctx, existing.ClassID, principal); err != nil {
		return err
	}

	if existing.Status == "COMPLETED" {
		return ErrSessionClosed
	}
	if existing.Status == "CANCELLED" {
		return ErrSessionCancelled
	}

	return s.repo.CloseSession(ctx, sessionID)
}

func (s *service) GetSessionRoster(ctx context.Context, sessionID uuid.UUID, principal *auth.Principal) (*SessionRosterResponse, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if err := s.authorizeClassManagement(ctx, session.ClassID, principal); err != nil {
		return nil, err
	}

	students, err := s.repo.GetSessionStudents(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	recordedCount := 0
	for _, st := range students {
		if st.AttendanceID != nil {
			recordedCount++
		}
	}

	return &SessionRosterResponse{
		SessionID:        session.ID,
		SessionNumber:    session.SessionNumber,
		ClassID:          session.ClassID,
		TotalStudents:    len(students),
		RecordedStudents: recordedCount,
		Students:         students,
	}, nil
}

func (s *service) GetSessionAttendance(ctx context.Context, sessionID uuid.UUID, principal *auth.Principal) ([]AttendanceDetailResponse, AttendanceSummary, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, AttendanceSummary{}, err
	}

	if err := s.authorizeClassManagement(ctx, session.ClassID, principal); err != nil {
		return nil, AttendanceSummary{}, err
	}

	attendances, summary, err := s.repo.GetSessionAttendances(ctx, sessionID)
	if err != nil {
		return nil, AttendanceSummary{}, err
	}

	responses := make([]AttendanceDetailResponse, len(attendances))
	for i, a := range attendances {
		responses[i] = AttendanceDetailResponse{
			ID:             a.ID,
			SessionID:      a.SessionID,
			StudentID:      a.StudentID,
			StudentNIM:     a.StudentNIM,
			StudentName:    a.StudentName,
			EnrollmentID:   a.EnrollmentID,
			Status:         a.Status,
			Remarks:        a.Remarks,
			RecordedByName: a.RecordedByName,
			RecordedAt:     a.RecordedAt.Format(time.RFC3339),
			CreatedAt:      a.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      a.UpdatedAt.Format(time.RFC3339),
		}
	}

	return responses, summary, nil
}

func (s *service) RecordBatchAttendance(ctx context.Context, sessionID uuid.UUID, req RecordBatchAttendanceRequest, principal *auth.Principal) error {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	if err := s.authorizeClassManagement(ctx, session.ClassID, principal); err != nil {
		return err
	}

	if session.Status == "CANCELLED" {
		return ErrSessionCancelled
	}

	if len(req.Items) == 0 {
		return validationError("items", "Daftar presensi mahasiswa tidak boleh kosong")
	}
	if len(req.Items) > MaxBatchAttendanceItems {
		return validationError("items", fmt.Sprintf("Jumlah mahasiswa dalam satu batch tidak boleh melebihi %d", MaxBatchAttendanceItems))
	}

	// Validate presence statuses and ensure no duplicate student_ids in payload
	seen := make(map[uuid.UUID]bool, len(req.Items))
	validStatuses := map[string]bool{
		"PRESENT": true,
		"LATE":    true,
		"EXCUSED": true,
		"SICK":    true,
		"ABSENT":  true,
	}

	for _, item := range req.Items {
		if seen[item.StudentID] {
			return appErrors.ErrConflict(fmt.Sprintf("Terdapat duplikasi ID mahasiswa '%s' dalam permintaan presensi", item.StudentID))
		}
		seen[item.StudentID] = true

		st := strings.ToUpper(strings.TrimSpace(item.Status))
		if !validStatuses[st] {
			return validationError("status", fmt.Sprintf("Status presensi '%s' tidak valid. Pilihan: PRESENT, LATE, EXCUSED, SICK, ABSENT", item.Status))
		}
	}

	return s.repo.RecordBatchAttendanceTx(ctx, sessionID, principal.UserID, req.Items)
}

func (s *service) UpdateAttendance(ctx context.Context, sessionID, attendanceID uuid.UUID, req UpdateAttendanceRequest, principal *auth.Principal) error {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	if err := s.authorizeClassManagement(ctx, session.ClassID, principal); err != nil {
		return err
	}

	st := strings.ToUpper(strings.TrimSpace(req.Status))
	switch st {
	case "PRESENT", "LATE", "EXCUSED", "SICK", "ABSENT":
		// valid
	default:
		return validationError("status", "Status presensi tidak valid. Pilihan: PRESENT, LATE, EXCUSED, SICK, ABSENT")
	}

	return s.repo.UpdateAttendanceTx(ctx, sessionID, attendanceID, principal.UserID, st, req.Remarks)
}

func (s *service) GetStudentAttendanceHistory(ctx context.Context, classID *uuid.UUID, p pagination.Params, principal *auth.Principal) ([]StudentAttendanceHistory, pagination.Meta, AttendanceSummary, error) {
	if principal == nil {
		return nil, pagination.Meta{}, AttendanceSummary{}, appErrors.ErrUnauthorized("Autentikasi diperlukan")
	}

	student, err := s.repo.GetStudentByUserID(ctx, principal.UserID)
	if err != nil {
		return nil, pagination.Meta{}, AttendanceSummary{}, err
	}

	if classID != nil {
		isEnrolled, err := s.repo.CheckStudentEnrolledInClass(ctx, *classID, student.ID)
		if err != nil {
			return nil, pagination.Meta{}, AttendanceSummary{}, err
		}
		if !isEnrolled {
			return nil, pagination.Meta{}, AttendanceSummary{}, appErrors.ErrForbidden("Anda tidak terdaftar sebagai peserta pada kelas ini")
		}
	}

	list, total, summary, err := s.repo.GetStudentAttendanceHistory(ctx, student.ID, classID, p)
	if err != nil {
		return nil, pagination.Meta{}, AttendanceSummary{}, err
	}

	meta := pagination.NewMeta(p.Page, p.Limit, total)
	return list, meta, summary, nil
}

func (s *service) GetClassAttendanceSummary(ctx context.Context, classID uuid.UUID, principal *auth.Principal) (*ClassAttendanceSummary, error) {
	if err := s.authorizeClassManagement(ctx, classID, principal); err != nil {
		return nil, err
	}

	return s.repo.GetClassAttendanceSummary(ctx, classID)
}

func mapSessionDetailToResponse(d *ClassSessionDetail) SessionResponse {
	return SessionResponse{
		ID:                d.ID,
		ClassID:           d.ClassID,
		ClassCode:         d.ClassCode,
		ClassName:         d.ClassName,
		CourseCode:        d.CourseCode,
		CourseName:        d.CourseName,
		SessionNumber:     d.SessionNumber,
		Date:              d.Date.Format("2006-01-02"),
		StartTime:         d.StartTime,
		EndTime:           d.EndTime,
		Topic:             d.Topic,
		Remarks:           d.Remarks,
		LecturerID:        d.LecturerID,
		LecturerName:      d.LecturerName,
		LecturerNIDN:      d.LecturerNIDN,
		Status:            d.Status,
		AttendanceSummary: d.AttendanceSummary,
		CreatedAt:         d.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         d.UpdatedAt.Format(time.RFC3339),
	}
}
