package attendance_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"educore/internal/modules/attendance"
	"educore/internal/modules/auth"
	"educore/internal/modules/lecturers"
	"educore/internal/modules/students"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
)

type mockRepository struct {
	checkLecturerTeachingClassFunc  func(ctx context.Context, classID, userID uuid.UUID) (bool, error)
	getLecturerByUserIDFunc         func(ctx context.Context, userID uuid.UUID) (*lecturers.Lecturer, error)
	getStudentByUserIDFunc          func(ctx context.Context, userID uuid.UUID) (*students.Student, error)
	checkStudentEnrolledInClassFunc func(ctx context.Context, classID, studentID uuid.UUID) (bool, error)
	createSessionFunc               func(ctx context.Context, s *attendance.ClassSession) error
	getSessionByIDFunc              func(ctx context.Context, sessionID uuid.UUID) (*attendance.ClassSessionDetail, error)
	listSessionsByClassFunc         func(ctx context.Context, classID uuid.UUID, p pagination.Params) ([]attendance.ClassSessionDetail, int, error)
	updateSessionFunc               func(ctx context.Context, s *attendance.ClassSession) error
	closeSessionFunc                func(ctx context.Context, sessionID uuid.UUID) error
	getSessionStudentsFunc          func(ctx context.Context, sessionID uuid.UUID) ([]attendance.SessionStudent, error)
	getSessionAttendancesFunc       func(ctx context.Context, sessionID uuid.UUID) ([]attendance.AttendanceDetail, attendance.AttendanceSummary, error)
	recordBatchAttendanceTxFunc     func(ctx context.Context, sessionID, recordedBy uuid.UUID, items []attendance.AttendanceItem) error
	updateAttendanceTxFunc          func(ctx context.Context, sessionID, attendanceID, recordedBy uuid.UUID, status string, remarks *string) error
	getStudentAttendanceHistoryFunc func(ctx context.Context, studentID uuid.UUID, classID *uuid.UUID, p pagination.Params) ([]attendance.StudentAttendanceHistory, int, attendance.AttendanceSummary, error)
	getClassAttendanceSummaryFunc   func(ctx context.Context, classID uuid.UUID) (*attendance.ClassAttendanceSummary, error)
}

func (m *mockRepository) CheckLecturerTeachingClass(ctx context.Context, classID, userID uuid.UUID) (bool, error) {
	if m.checkLecturerTeachingClassFunc != nil {
		return m.checkLecturerTeachingClassFunc(ctx, classID, userID)
	}
	return true, nil
}

func (m *mockRepository) GetLecturerByUserID(ctx context.Context, userID uuid.UUID) (*lecturers.Lecturer, error) {
	if m.getLecturerByUserIDFunc != nil {
		return m.getLecturerByUserIDFunc(ctx, userID)
	}
	return &lecturers.Lecturer{ID: uuid.New(), UserID: &userID, Name: "Dosen Pengampu"}, nil
}

func (m *mockRepository) GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*students.Student, error) {
	if m.getStudentByUserIDFunc != nil {
		return m.getStudentByUserIDFunc(ctx, userID)
	}
	return &students.Student{ID: uuid.New(), UserID: &userID, NIM: "2024001", Name: "Mahasiswa Test"}, nil
}

func (m *mockRepository) CheckStudentEnrolledInClass(ctx context.Context, classID, studentID uuid.UUID) (bool, error) {
	if m.checkStudentEnrolledInClassFunc != nil {
		return m.checkStudentEnrolledInClassFunc(ctx, classID, studentID)
	}
	return true, nil
}

func (m *mockRepository) CreateSession(ctx context.Context, s *attendance.ClassSession) error {
	if m.createSessionFunc != nil {
		return m.createSessionFunc(ctx, s)
	}
	return nil
}

func (m *mockRepository) GetSessionByID(ctx context.Context, sessionID uuid.UUID) (*attendance.ClassSessionDetail, error) {
	if m.getSessionByIDFunc != nil {
		return m.getSessionByIDFunc(ctx, sessionID)
	}
	return &attendance.ClassSessionDetail{
		ClassSession: attendance.ClassSession{
			ID:            sessionID,
			ClassID:       uuid.New(),
			SessionNumber: 1,
			Date:          time.Now(),
			StartTime:     "08:00:00",
			EndTime:       "09:40:00",
			Topic:         "Pengantar Pemrograman",
			Status:        "OPEN",
		},
		ClassCode:  "IF101-A",
		ClassName:  "Pemrograman Dasar A",
		CourseCode: "IF101",
		CourseName: "Pemrograman Dasar",
	}, nil
}

func (m *mockRepository) ListSessionsByClass(ctx context.Context, classID uuid.UUID, p pagination.Params) ([]attendance.ClassSessionDetail, int, error) {
	if m.listSessionsByClassFunc != nil {
		return m.listSessionsByClassFunc(ctx, classID, p)
	}
	return nil, 0, nil
}

func (m *mockRepository) UpdateSession(ctx context.Context, s *attendance.ClassSession) error {
	if m.updateSessionFunc != nil {
		return m.updateSessionFunc(ctx, s)
	}
	return nil
}

func (m *mockRepository) CloseSession(ctx context.Context, sessionID uuid.UUID) error {
	if m.closeSessionFunc != nil {
		return m.closeSessionFunc(ctx, sessionID)
	}
	return nil
}

func (m *mockRepository) GetSessionStudents(ctx context.Context, sessionID uuid.UUID) ([]attendance.SessionStudent, error) {
	if m.getSessionStudentsFunc != nil {
		return m.getSessionStudentsFunc(ctx, sessionID)
	}
	return nil, nil
}

func (m *mockRepository) GetSessionAttendances(ctx context.Context, sessionID uuid.UUID) ([]attendance.AttendanceDetail, attendance.AttendanceSummary, error) {
	if m.getSessionAttendancesFunc != nil {
		return m.getSessionAttendancesFunc(ctx, sessionID)
	}
	return nil, attendance.AttendanceSummary{}, nil
}

func (m *mockRepository) RecordBatchAttendanceTx(ctx context.Context, sessionID, recordedBy uuid.UUID, items []attendance.AttendanceItem) error {
	if m.recordBatchAttendanceTxFunc != nil {
		return m.recordBatchAttendanceTxFunc(ctx, sessionID, recordedBy, items)
	}
	return nil
}

func (m *mockRepository) UpdateAttendanceTx(ctx context.Context, sessionID, attendanceID, recordedBy uuid.UUID, status string, remarks *string) error {
	if m.updateAttendanceTxFunc != nil {
		return m.updateAttendanceTxFunc(ctx, sessionID, attendanceID, recordedBy, status, remarks)
	}
	return nil
}

func (m *mockRepository) GetStudentAttendanceHistory(ctx context.Context, studentID uuid.UUID, classID *uuid.UUID, p pagination.Params) ([]attendance.StudentAttendanceHistory, int, attendance.AttendanceSummary, error) {
	if m.getStudentAttendanceHistoryFunc != nil {
		return m.getStudentAttendanceHistoryFunc(ctx, studentID, classID, p)
	}
	return nil, 0, attendance.AttendanceSummary{}, nil
}

func (m *mockRepository) GetClassAttendanceSummary(ctx context.Context, classID uuid.UUID) (*attendance.ClassAttendanceSummary, error) {
	if m.getClassAttendanceSummaryFunc != nil {
		return m.getClassAttendanceSummaryFunc(ctx, classID)
	}
	return &attendance.ClassAttendanceSummary{ClassID: classID}, nil
}

func setupTestService(repo attendance.Repository) attendance.Service {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return attendance.NewService(repo, logger)
}

func TestCreateSession_Success(t *testing.T) {
	classID := uuid.New()
	lecturerUserID := uuid.New()
	lecturerID := uuid.New()

	repo := &mockRepository{
		checkLecturerTeachingClassFunc: func(ctx context.Context, cid, uid uuid.UUID) (bool, error) {
			if cid != classID || uid != lecturerUserID {
				t.Fatalf("unexpected arguments: cid=%v, uid=%v", cid, uid)
			}
			return true, nil
		},
		getLecturerByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*lecturers.Lecturer, error) {
			return &lecturers.Lecturer{ID: lecturerID, UserID: &uid, Name: "Dr. Budi"}, nil
		},
		createSessionFunc: func(ctx context.Context, s *attendance.ClassSession) error {
			if s.ClassID != classID {
				t.Fatalf("expected ClassID=%v, got %v", classID, s.ClassID)
			}
			if s.SessionNumber != 1 {
				t.Fatalf("expected SessionNumber=1, got %d", s.SessionNumber)
			}
			if s.Topic != "Pengantar Algoritma" {
				t.Fatalf("expected Topic='Pengantar Algoritma', got %s", s.Topic)
			}
			if s.Status != "OPEN" {
				t.Fatalf("expected Status='OPEN', got %s", s.Status)
			}
			return nil
		},
		getSessionByIDFunc: func(ctx context.Context, sessionID uuid.UUID) (*attendance.ClassSessionDetail, error) {
			return &attendance.ClassSessionDetail{
				ClassSession: attendance.ClassSession{
					ID:            sessionID,
					ClassID:       classID,
					SessionNumber: 1,
					Date:          time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
					StartTime:     "08:00",
					EndTime:       "09:40",
					Topic:         "Pengantar Algoritma",
					Status:        "OPEN",
				},
			}, nil
		},
	}

	svc := setupTestService(repo)

	principal := &auth.Principal{
		UserID: lecturerUserID,
		Roles:  []string{"dosen"},
	}

	req := attendance.CreateSessionRequest{
		SessionNumber: 1,
		Date:          "2026-10-15",
		StartTime:     "08:00",
		EndTime:       "09:40",
		Topic:         "Pengantar Algoritma",
	}

	resp, err := svc.CreateSession(context.Background(), classID, req, principal)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected resp not nil")
	}
	if resp.SessionNumber != 1 || resp.Topic != "Pengantar Algoritma" || resp.Status != "OPEN" {
		t.Fatalf("unexpected response content: %+v", resp)
	}
}

func TestCreateSession_UnauthorizedLecturer(t *testing.T) {
	classID := uuid.New()
	lecturerUserID := uuid.New()

	repo := &mockRepository{
		checkLecturerTeachingClassFunc: func(ctx context.Context, cid, uid uuid.UUID) (bool, error) {
			return false, nil // Not assigned to this class
		},
	}

	svc := setupTestService(repo)

	principal := &auth.Principal{
		UserID: lecturerUserID,
		Roles:  []string{"dosen"},
	}

	req := attendance.CreateSessionRequest{
		SessionNumber: 1,
		Date:          "2026-10-15",
		StartTime:     "08:00",
		EndTime:       "09:40",
		Topic:         "Pengantar Algoritma",
	}

	_, err := svc.CreateSession(context.Background(), classID, req, principal)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, attendance.ErrUnauthorizedLecturer) {
		t.Fatalf("expected ErrUnauthorizedLecturer, got: %v", err)
	}
}

func TestCreateSession_AdminBypass(t *testing.T) {
	classID := uuid.New()
	adminUserID := uuid.New()

	repo := &mockRepository{
		createSessionFunc: func(ctx context.Context, s *attendance.ClassSession) error {
			return nil
		},
		getSessionByIDFunc: func(ctx context.Context, sessionID uuid.UUID) (*attendance.ClassSessionDetail, error) {
			return &attendance.ClassSessionDetail{
				ClassSession: attendance.ClassSession{
					ID:            sessionID,
					ClassID:       classID,
					SessionNumber: 1,
					Date:          time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
					StartTime:     "08:00",
					EndTime:       "09:40",
					Topic:         "Pengantar Algoritma",
					Status:        "OPEN",
				},
			}, nil
		},
	}

	svc := setupTestService(repo)

	principal := &auth.Principal{
		UserID: adminUserID,
		Roles:  []string{"admin"},
	}

	req := attendance.CreateSessionRequest{
		SessionNumber: 1,
		Date:          "2026-10-15",
		StartTime:     "08:00",
		EndTime:       "09:40",
		Topic:         "Pengantar Algoritma",
	}

	resp, err := svc.CreateSession(context.Background(), classID, req, principal)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected resp not nil")
	}
}

func TestCreateSession_InvalidTimeRange(t *testing.T) {
	classID := uuid.New()
	adminUserID := uuid.New()

	repo := &mockRepository{}
	svc := setupTestService(repo)

	principal := &auth.Principal{
		UserID: adminUserID,
		Roles:  []string{"admin"},
	}

	req := attendance.CreateSessionRequest{
		SessionNumber: 1,
		Date:          "2026-10-15",
		StartTime:     "10:00",
		EndTime:       "08:00",
		Topic:         "Pengantar Algoritma",
	}

	_, err := svc.CreateSession(context.Background(), classID, req, principal)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *appErrors.AppError, got %T: %v", err, err)
	}
	if appErr.Code != appErrors.CodeValidationError {
		t.Fatalf("expected CodeValidationError, got %s", appErr.Code)
	}
}

func TestCreateSession_InvalidDateFormat(t *testing.T) {
	classID := uuid.New()
	adminUserID := uuid.New()

	repo := &mockRepository{}
	svc := setupTestService(repo)

	principal := &auth.Principal{
		UserID: adminUserID,
		Roles:  []string{"admin"},
	}

	req := attendance.CreateSessionRequest{
		SessionNumber: 1,
		Date:          "15-10-2026", // Invalid format, must be YYYY-MM-DD
		StartTime:     "08:00",
		EndTime:       "09:40",
		Topic:         "Pengantar Algoritma",
	}

	_, err := svc.CreateSession(context.Background(), classID, req, principal)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *appErrors.AppError, got %T: %v", err, err)
	}
	if appErr.Code != appErrors.CodeValidationError {
		t.Fatalf("expected CodeValidationError, got %s", appErr.Code)
	}
}

func TestCreateSession_DuplicateSessionNumber(t *testing.T) {
	classID := uuid.New()
	adminUserID := uuid.New()

	repo := &mockRepository{
		createSessionFunc: func(ctx context.Context, s *attendance.ClassSession) error {
			return attendance.ErrDuplicateSessionNumber
		},
	}

	svc := setupTestService(repo)

	principal := &auth.Principal{
		UserID: adminUserID,
		Roles:  []string{"admin"},
	}

	req := attendance.CreateSessionRequest{
		SessionNumber: 1,
		Date:          "2026-10-15",
		StartTime:     "08:00",
		EndTime:       "09:40",
		Topic:         "Pengantar Algoritma",
	}

	_, err := svc.CreateSession(context.Background(), classID, req, principal)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, attendance.ErrDuplicateSessionNumber) {
		t.Fatalf("expected ErrDuplicateSessionNumber, got: %v", err)
	}
}

func TestGetSessionDetail_StudentAccess(t *testing.T) {
	sessionID := uuid.New()
	classID := uuid.New()
	studentUserID := uuid.New()
	studentID := uuid.New()

	t.Run("Allowed when student is enrolled", func(t *testing.T) {
		repo := &mockRepository{
			getSessionByIDFunc: func(ctx context.Context, sid uuid.UUID) (*attendance.ClassSessionDetail, error) {
				return &attendance.ClassSessionDetail{
					ClassSession: attendance.ClassSession{ID: sid, ClassID: classID},
				}, nil
			},
			getStudentByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*students.Student, error) {
				return &students.Student{ID: studentID, UserID: &uid}, nil
			},
			checkStudentEnrolledInClassFunc: func(ctx context.Context, cid, sid uuid.UUID) (bool, error) {
				return true, nil
			},
		}

		svc := setupTestService(repo)
		principal := &auth.Principal{UserID: studentUserID, Roles: []string{"mahasiswa"}}

		resp, err := svc.GetSessionDetail(context.Background(), sessionID, principal)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if resp.ID != sessionID {
			t.Fatalf("expected session ID %v, got %v", sessionID, resp.ID)
		}
	})

	t.Run("Forbidden when student is not enrolled", func(t *testing.T) {
		repo := &mockRepository{
			getSessionByIDFunc: func(ctx context.Context, sid uuid.UUID) (*attendance.ClassSessionDetail, error) {
				return &attendance.ClassSessionDetail{
					ClassSession: attendance.ClassSession{ID: sid, ClassID: classID},
				}, nil
			},
			getStudentByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*students.Student, error) {
				return &students.Student{ID: studentID, UserID: &uid}, nil
			},
			checkStudentEnrolledInClassFunc: func(ctx context.Context, cid, sid uuid.UUID) (bool, error) {
				return false, nil // Not enrolled
			},
		}

		svc := setupTestService(repo)
		principal := &auth.Principal{UserID: studentUserID, Roles: []string{"mahasiswa"}}

		_, err := svc.GetSessionDetail(context.Background(), sessionID, principal)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var appErr *appErrors.AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected *appErrors.AppError, got %T: %v", err, err)
		}
		if appErr.Code != appErrors.CodeForbidden {
			t.Fatalf("expected CodeForbidden, got %s", appErr.Code)
		}
	})
}

func TestCloseSession_StatusChecks(t *testing.T) {
	sessionID := uuid.New()
	classID := uuid.New()
	adminUserID := uuid.New()
	principal := &auth.Principal{UserID: adminUserID, Roles: []string{"admin"}}

	t.Run("Success when session is open", func(t *testing.T) {
		repo := &mockRepository{
			getSessionByIDFunc: func(ctx context.Context, sid uuid.UUID) (*attendance.ClassSessionDetail, error) {
				return &attendance.ClassSessionDetail{
					ClassSession: attendance.ClassSession{ID: sid, ClassID: classID, Status: "OPEN"},
				}, nil
			},
			closeSessionFunc: func(ctx context.Context, sid uuid.UUID) error {
				return nil
			},
		}
		svc := setupTestService(repo)
		err := svc.CloseSession(context.Background(), sessionID, principal)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})

	t.Run("Conflict when session already completed", func(t *testing.T) {
		repo := &mockRepository{
			getSessionByIDFunc: func(ctx context.Context, sid uuid.UUID) (*attendance.ClassSessionDetail, error) {
				return &attendance.ClassSessionDetail{
					ClassSession: attendance.ClassSession{ID: sid, ClassID: classID, Status: "COMPLETED"},
				}, nil
			},
		}
		svc := setupTestService(repo)
		err := svc.CloseSession(context.Background(), sessionID, principal)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, attendance.ErrSessionClosed) {
			t.Fatalf("expected ErrSessionClosed, got: %v", err)
		}
	})

	t.Run("Conflict when session cancelled", func(t *testing.T) {
		repo := &mockRepository{
			getSessionByIDFunc: func(ctx context.Context, sid uuid.UUID) (*attendance.ClassSessionDetail, error) {
				return &attendance.ClassSessionDetail{
					ClassSession: attendance.ClassSession{ID: sid, ClassID: classID, Status: "CANCELLED"},
				}, nil
			},
		}
		svc := setupTestService(repo)
		err := svc.CloseSession(context.Background(), sessionID, principal)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, attendance.ErrSessionCancelled) {
			t.Fatalf("expected ErrSessionCancelled, got: %v", err)
		}
	})
}

func TestRecordBatchAttendance_Validation(t *testing.T) {
	sessionID := uuid.New()
	classID := uuid.New()
	adminUserID := uuid.New()
	principal := &auth.Principal{UserID: adminUserID, Roles: []string{"admin"}}

	t.Run("Rejects duplicate students in batch payload", func(t *testing.T) {
		repo := &mockRepository{
			getSessionByIDFunc: func(ctx context.Context, sid uuid.UUID) (*attendance.ClassSessionDetail, error) {
				return &attendance.ClassSessionDetail{
					ClassSession: attendance.ClassSession{ID: sid, ClassID: classID, Status: "OPEN"},
				}, nil
			},
		}
		svc := setupTestService(repo)

		dupStudentID := uuid.New()
		req := attendance.RecordBatchAttendanceRequest{
			Items: []attendance.AttendanceItem{
				{StudentID: dupStudentID, Status: "PRESENT"},
				{StudentID: dupStudentID, Status: "LATE"},
			},
		}

		err := svc.RecordBatchAttendance(context.Background(), sessionID, req, principal)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var appErr *appErrors.AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected *appErrors.AppError, got %T: %v", err, err)
		}
		if appErr.Code != appErrors.CodeConflict {
			t.Fatalf("expected CodeConflict, got %s", appErr.Code)
		}
	})

	t.Run("Rejects invalid attendance status", func(t *testing.T) {
		repo := &mockRepository{
			getSessionByIDFunc: func(ctx context.Context, sid uuid.UUID) (*attendance.ClassSessionDetail, error) {
				return &attendance.ClassSessionDetail{
					ClassSession: attendance.ClassSession{ID: sid, ClassID: classID, Status: "OPEN"},
				}, nil
			},
		}
		svc := setupTestService(repo)

		req := attendance.RecordBatchAttendanceRequest{
			Items: []attendance.AttendanceItem{
				{StudentID: uuid.New(), Status: "UNKNOWN_STATUS"},
			},
		}

		err := svc.RecordBatchAttendance(context.Background(), sessionID, req, principal)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var appErr *appErrors.AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected *appErrors.AppError, got %T: %v", err, err)
		}
		if appErr.Code != appErrors.CodeValidationError {
			t.Fatalf("expected CodeValidationError, got %s", appErr.Code)
		}
	})

	t.Run("Succeeds with valid items", func(t *testing.T) {
		repo := &mockRepository{
			getSessionByIDFunc: func(ctx context.Context, sid uuid.UUID) (*attendance.ClassSessionDetail, error) {
				return &attendance.ClassSessionDetail{
					ClassSession: attendance.ClassSession{ID: sid, ClassID: classID, Status: "OPEN"},
				}, nil
			},
			recordBatchAttendanceTxFunc: func(ctx context.Context, sid, recordedBy uuid.UUID, items []attendance.AttendanceItem) error {
				if sid != sessionID || recordedBy != adminUserID {
					t.Fatalf("unexpected sid/recordedBy: sid=%v, rec=%v", sid, recordedBy)
				}
				if len(items) != 2 {
					t.Fatalf("expected 2 items, got %d", len(items))
				}
				return nil
			},
		}
		svc := setupTestService(repo)

		req := attendance.RecordBatchAttendanceRequest{
			Items: []attendance.AttendanceItem{
				{StudentID: uuid.New(), Status: "PRESENT"},
				{StudentID: uuid.New(), Status: "EXCUSED"},
			},
		}

		err := svc.RecordBatchAttendance(context.Background(), sessionID, req, principal)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})
}

func TestUpdateAttendance_Validation(t *testing.T) {
	sessionID := uuid.New()
	attendanceID := uuid.New()
	classID := uuid.New()
	adminUserID := uuid.New()
	principal := &auth.Principal{UserID: adminUserID, Roles: []string{"admin"}}

	t.Run("Rejects invalid status", func(t *testing.T) {
		repo := &mockRepository{
			getSessionByIDFunc: func(ctx context.Context, sid uuid.UUID) (*attendance.ClassSessionDetail, error) {
				return &attendance.ClassSessionDetail{
					ClassSession: attendance.ClassSession{ID: sid, ClassID: classID, Status: "OPEN"},
				}, nil
			},
		}
		svc := setupTestService(repo)

		req := attendance.UpdateAttendanceRequest{
			Status: "INVALID",
		}

		err := svc.UpdateAttendance(context.Background(), sessionID, attendanceID, req, principal)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var appErr *appErrors.AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected *appErrors.AppError, got %T: %v", err, err)
		}
		if appErr.Code != appErrors.CodeValidationError {
			t.Fatalf("expected CodeValidationError, got %s", appErr.Code)
		}
	})

	t.Run("Succeeds with valid status", func(t *testing.T) {
		repo := &mockRepository{
			getSessionByIDFunc: func(ctx context.Context, sid uuid.UUID) (*attendance.ClassSessionDetail, error) {
				return &attendance.ClassSessionDetail{
					ClassSession: attendance.ClassSession{ID: sid, ClassID: classID, Status: "OPEN"},
				}, nil
			},
			updateAttendanceTxFunc: func(ctx context.Context, sid, aid, recordedBy uuid.UUID, status string, remarks *string) error {
				if sid != sessionID || aid != attendanceID {
					t.Fatalf("unexpected sid/aid: sid=%v, aid=%v", sid, aid)
				}
				if status != "SICK" {
					t.Fatalf("expected status 'SICK', got %s", status)
				}
				return nil
			},
		}
		svc := setupTestService(repo)

		req := attendance.UpdateAttendanceRequest{
			Status: "SICK",
		}

		err := svc.UpdateAttendance(context.Background(), sessionID, attendanceID, req, principal)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})
}
