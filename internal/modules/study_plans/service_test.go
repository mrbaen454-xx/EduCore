package study_plans_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"educore/internal/modules/auth"
	"educore/internal/modules/students"
	"educore/internal/modules/study_plans"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
)

type mockRepository struct {
	getStudentByUserIDFunc          func(ctx context.Context, userID uuid.UUID) (*students.Student, error)
	getStudentByIDFunc              func(ctx context.Context, studentID uuid.UUID) (*students.Student, error)
	checkAcademicPeriodActiveFunc   func(ctx context.Context, periodID uuid.UUID) (bool, bool, error)
	getActiveAcademicPeriodIDFunc   func(ctx context.Context) (uuid.UUID, error)
	createStudyPlanFunc             func(ctx context.Context, sp *study_plans.StudyPlan) error
	getStudyPlanByIDFunc            func(ctx context.Context, id uuid.UUID) (*study_plans.StudyPlanDetail, error)
	getStudyPlanByStudentPeriodFunc func(ctx context.Context, studentID, periodID uuid.UUID) (*study_plans.StudyPlanDetail, error)
	listStudyPlansFunc              func(ctx context.Context, filter study_plans.StudyPlanFilter, p pagination.Params) ([]study_plans.StudyPlanDetail, int, error)
	listAvailableClassesFunc        func(ctx context.Context, filter study_plans.AvailableClassFilter, p pagination.Params) ([]study_plans.AvailableClass, int, error)
	getEnrollmentsByStudyPlanFunc   func(ctx context.Context, studyPlanID uuid.UUID) ([]study_plans.EnrollmentDetail, error)
	getEnrollmentByIDFunc           func(ctx context.Context, enrollmentID uuid.UUID) (*study_plans.EnrollmentDetail, error)
	enrollClassTxFunc               func(ctx context.Context, studyPlanID, studentID, classID uuid.UUID, bypassOwnership bool) (*study_plans.EnrollmentDetail, error)
	dropEnrollmentTxFunc            func(ctx context.Context, studyPlanID, studentID, enrollmentID uuid.UUID, bypassOwnership bool) error
}

func (m *mockRepository) GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*students.Student, error) {
	if m.getStudentByUserIDFunc != nil {
		return m.getStudentByUserIDFunc(ctx, userID)
	}
	return nil, study_plans.ErrStudentNotFound
}

func (m *mockRepository) GetStudentByID(ctx context.Context, studentID uuid.UUID) (*students.Student, error) {
	if m.getStudentByIDFunc != nil {
		return m.getStudentByIDFunc(ctx, studentID)
	}
	return nil, study_plans.ErrStudentNotFound
}

func (m *mockRepository) CheckAcademicPeriodActive(ctx context.Context, periodID uuid.UUID) (bool, bool, error) {
	if m.checkAcademicPeriodActiveFunc != nil {
		return m.checkAcademicPeriodActiveFunc(ctx, periodID)
	}
	return true, true, nil
}

func (m *mockRepository) GetActiveAcademicPeriodID(ctx context.Context) (uuid.UUID, error) {
	if m.getActiveAcademicPeriodIDFunc != nil {
		return m.getActiveAcademicPeriodIDFunc(ctx)
	}
	return uuid.New(), nil
}

func (m *mockRepository) CreateStudyPlan(ctx context.Context, sp *study_plans.StudyPlan) error {
	if m.createStudyPlanFunc != nil {
		return m.createStudyPlanFunc(ctx, sp)
	}
	sp.ID = uuid.New()
	return nil
}

func (m *mockRepository) GetStudyPlanByID(ctx context.Context, id uuid.UUID) (*study_plans.StudyPlanDetail, error) {
	if m.getStudyPlanByIDFunc != nil {
		return m.getStudyPlanByIDFunc(ctx, id)
	}
	return nil, study_plans.ErrStudyPlanNotFound
}

func (m *mockRepository) GetStudyPlanByStudentAndPeriod(ctx context.Context, studentID, periodID uuid.UUID) (*study_plans.StudyPlanDetail, error) {
	if m.getStudyPlanByStudentPeriodFunc != nil {
		return m.getStudyPlanByStudentPeriodFunc(ctx, studentID, periodID)
	}
	return nil, study_plans.ErrStudyPlanNotFound
}

func (m *mockRepository) ListStudyPlans(ctx context.Context, filter study_plans.StudyPlanFilter, p pagination.Params) ([]study_plans.StudyPlanDetail, int, error) {
	if m.listStudyPlansFunc != nil {
		return m.listStudyPlansFunc(ctx, filter, p)
	}
	return nil, 0, nil
}

func (m *mockRepository) ListAvailableClasses(ctx context.Context, filter study_plans.AvailableClassFilter, p pagination.Params) ([]study_plans.AvailableClass, int, error) {
	if m.listAvailableClassesFunc != nil {
		return m.listAvailableClassesFunc(ctx, filter, p)
	}
	return nil, 0, nil
}

func (m *mockRepository) GetEnrollmentsByStudyPlan(ctx context.Context, studyPlanID uuid.UUID) ([]study_plans.EnrollmentDetail, error) {
	if m.getEnrollmentsByStudyPlanFunc != nil {
		return m.getEnrollmentsByStudyPlanFunc(ctx, studyPlanID)
	}
	return nil, nil
}

func (m *mockRepository) GetEnrollmentByID(ctx context.Context, enrollmentID uuid.UUID) (*study_plans.EnrollmentDetail, error) {
	if m.getEnrollmentByIDFunc != nil {
		return m.getEnrollmentByIDFunc(ctx, enrollmentID)
	}
	return nil, study_plans.ErrEnrollmentNotFound
}

func (m *mockRepository) EnrollClassTx(ctx context.Context, studyPlanID, studentID, classID uuid.UUID, bypassOwnership bool) (*study_plans.EnrollmentDetail, error) {
	if m.enrollClassTxFunc != nil {
		return m.enrollClassTxFunc(ctx, studyPlanID, studentID, classID, bypassOwnership)
	}
	return &study_plans.EnrollmentDetail{
		Enrollment: study_plans.Enrollment{
			ID:          uuid.New(),
			StudyPlanID: studyPlanID,
			StudentID:   studentID,
			ClassID:     classID,
			Status:      "ENROLLED",
			EnrolledAt:  time.Now(),
		},
		CourseCredits: 3,
	}, nil
}

func (m *mockRepository) DropEnrollmentTx(ctx context.Context, studyPlanID, studentID, enrollmentID uuid.UUID, bypassOwnership bool) error {
	if m.dropEnrollmentTxFunc != nil {
		return m.dropEnrollmentTxFunc(ctx, studyPlanID, studentID, enrollmentID, bypassOwnership)
	}
	return nil
}

func setupTestService(mock *mockRepository) study_plans.Service {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	return study_plans.NewService(mock, logger)
}

func TestStudyPlans_StudentStatusValidation(t *testing.T) {
	studentID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name          string
		studentStatus string
		expectErr     bool
	}{
		{"Active student should pass", "ACTIVE", false},
		{"Inactive student should be rejected", "INACTIVE", true},
		{"Suspended student should be rejected", "SUSPENDED", true},
		{"Graduated student should be rejected", "GRADUATED", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockRepository{
				getStudentByUserIDFunc: func(ctx context.Context, uID uuid.UUID) (*students.Student, error) {
					return &students.Student{
						ID:     studentID,
						UserID: &userID,
						Status: tc.studentStatus,
					}, nil
				},
			}
			svc := setupTestService(mock)
			principal := &auth.Principal{UserID: userID, Roles: []string{"mahasiswa"}}

			student, err := svc.GetStudentProfile(context.Background(), principal)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error for status %s, got nil", tc.studentStatus)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error for status %s, got: %v", tc.studentStatus, err)
				}
				if student.ID != studentID {
					t.Errorf("expected student ID %v, got %v", studentID, student.ID)
				}
			}
		})
	}
}

func TestStudyPlans_GetOrCreateMyStudyPlan(t *testing.T) {
	studentID := uuid.New()
	userID := uuid.New()
	periodID := uuid.New()
	existingSPID := uuid.New()

	t.Run("Return existing study plan if already created", func(t *testing.T) {
		mock := &mockRepository{
			getStudentByUserIDFunc: func(ctx context.Context, uID uuid.UUID) (*students.Student, error) {
				return &students.Student{ID: studentID, UserID: &userID, Status: "ACTIVE"}, nil
			},
			checkAcademicPeriodActiveFunc: func(ctx context.Context, pID uuid.UUID) (bool, bool, error) {
				return true, true, nil
			},
			getStudyPlanByStudentPeriodFunc: func(ctx context.Context, sID, pID uuid.UUID) (*study_plans.StudyPlanDetail, error) {
				return &study_plans.StudyPlanDetail{
					StudyPlan: study_plans.StudyPlan{ID: existingSPID, StudentID: sID, AcademicPeriodID: pID, Status: "ACTIVE"},
				}, nil
			},
		}
		svc := setupTestService(mock)
		principal := &auth.Principal{UserID: userID, Roles: []string{"mahasiswa"}}

		sp, created, err := svc.GetOrCreateMyStudyPlan(context.Background(), principal, periodID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created {
			t.Errorf("expected created = false, got true")
		}
		if sp.ID != existingSPID {
			t.Errorf("expected sp ID %v, got %v", existingSPID, sp.ID)
		}
	})

	t.Run("Create new study plan if not existing", func(t *testing.T) {
		createdID := uuid.New()
		mock := &mockRepository{
			getStudentByUserIDFunc: func(ctx context.Context, uID uuid.UUID) (*students.Student, error) {
				return &students.Student{ID: studentID, UserID: &userID, Status: "ACTIVE"}, nil
			},
			checkAcademicPeriodActiveFunc: func(ctx context.Context, pID uuid.UUID) (bool, bool, error) {
				return true, true, nil
			},
			getStudyPlanByStudentPeriodFunc: func(ctx context.Context, sID, pID uuid.UUID) (*study_plans.StudyPlanDetail, error) {
				return nil, study_plans.ErrStudyPlanNotFound
			},
			createStudyPlanFunc: func(ctx context.Context, sp *study_plans.StudyPlan) error {
				sp.ID = createdID
				return nil
			},
			getStudyPlanByIDFunc: func(ctx context.Context, id uuid.UUID) (*study_plans.StudyPlanDetail, error) {
				return &study_plans.StudyPlanDetail{
					StudyPlan: study_plans.StudyPlan{ID: createdID, StudentID: studentID, AcademicPeriodID: periodID, Status: "ACTIVE"},
				}, nil
			},
		}
		svc := setupTestService(mock)
		principal := &auth.Principal{UserID: userID, Roles: []string{"mahasiswa"}}

		sp, created, err := svc.GetOrCreateMyStudyPlan(context.Background(), principal, periodID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !created {
			t.Errorf("expected created = true, got false")
		}
		if sp.ID != createdID {
			t.Errorf("expected sp ID %v, got %v", createdID, sp.ID)
		}
	})

	t.Run("Reject if academic period is inactive", func(t *testing.T) {
		mock := &mockRepository{
			getStudentByUserIDFunc: func(ctx context.Context, uID uuid.UUID) (*students.Student, error) {
				return &students.Student{ID: studentID, UserID: &userID, Status: "ACTIVE"}, nil
			},
			checkAcademicPeriodActiveFunc: func(ctx context.Context, pID uuid.UUID) (bool, bool, error) {
				return true, false, nil // Inactive!
			},
		}
		svc := setupTestService(mock)
		principal := &auth.Principal{UserID: userID, Roles: []string{"mahasiswa"}}

		_, _, err := svc.GetOrCreateMyStudyPlan(context.Background(), principal, periodID)
		if err != study_plans.ErrAcademicPeriodNotActive {
			t.Fatalf("expected ErrAcademicPeriodNotActive, got: %v", err)
		}
	})
}

func TestStudyPlans_OwnershipEnforcement(t *testing.T) {
	student1ID := uuid.New()
	student2ID := uuid.New()
	user1ID := uuid.New()
	spID := uuid.New()

	mock := &mockRepository{
		getStudentByUserIDFunc: func(ctx context.Context, uID uuid.UUID) (*students.Student, error) {
			return &students.Student{ID: student1ID, UserID: &user1ID, Status: "ACTIVE"}, nil
		},
		getStudyPlanByIDFunc: func(ctx context.Context, id uuid.UUID) (*study_plans.StudyPlanDetail, error) {
			return &study_plans.StudyPlanDetail{
				StudyPlan: study_plans.StudyPlan{ID: spID, StudentID: student2ID, Status: "ACTIVE"}, // Belongs to student 2!
			}, nil
		},
	}
	svc := setupTestService(mock)
	principal := &auth.Principal{UserID: user1ID, Roles: []string{"mahasiswa"}}

	_, err := svc.GetMyStudyPlanDetail(context.Background(), principal, spID)
	if err != study_plans.ErrUnauthorizedStudyPlan {
		t.Fatalf("expected ErrUnauthorizedStudyPlan, got %v", err)
	}
}

func TestStudyPlans_EnrollAndDropWorkflows(t *testing.T) {
	studentID := uuid.New()
	userID := uuid.New()
	spID := uuid.New()
	classID := uuid.New()
	enrollmentID := uuid.New()

	t.Run("Enroll class success", func(t *testing.T) {
		mock := &mockRepository{
			getStudentByUserIDFunc: func(ctx context.Context, uID uuid.UUID) (*students.Student, error) {
				return &students.Student{ID: studentID, UserID: &userID, Status: "ACTIVE"}, nil
			},
			enrollClassTxFunc: func(ctx context.Context, sPlanID, sID, cID uuid.UUID, bypass bool) (*study_plans.EnrollmentDetail, error) {
				return &study_plans.EnrollmentDetail{
					Enrollment: study_plans.Enrollment{
						ID:          enrollmentID,
						StudyPlanID: sPlanID,
						StudentID:   sID,
						ClassID:     cID,
						Status:      "ENROLLED",
					},
					CourseCredits: 3,
				}, nil
			},
		}
		svc := setupTestService(mock)
		principal := &auth.Principal{UserID: userID, Roles: []string{"mahasiswa"}}

		enrolled, err := svc.EnrollClassInMyStudyPlan(context.Background(), principal, spID, classID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if enrolled.ID != enrollmentID {
			t.Errorf("expected enrollment ID %v, got %v", enrollmentID, enrolled.ID)
		}
	})

	t.Run("Enroll rejected if capacity full", func(t *testing.T) {
		mock := &mockRepository{
			getStudentByUserIDFunc: func(ctx context.Context, uID uuid.UUID) (*students.Student, error) {
				return &students.Student{ID: studentID, UserID: &userID, Status: "ACTIVE"}, nil
			},
			enrollClassTxFunc: func(ctx context.Context, sPlanID, sID, cID uuid.UUID, bypass bool) (*study_plans.EnrollmentDetail, error) {
				return nil, study_plans.ErrClassCapacityFull
			},
		}
		svc := setupTestService(mock)
		principal := &auth.Principal{UserID: userID, Roles: []string{"mahasiswa"}}

		_, err := svc.EnrollClassInMyStudyPlan(context.Background(), principal, spID, classID)
		if err != study_plans.ErrClassCapacityFull {
			t.Fatalf("expected ErrClassCapacityFull, got: %v", err)
		}
	})

	t.Run("Enroll rejected if max 24 credits exceeded", func(t *testing.T) {
		mock := &mockRepository{
			getStudentByUserIDFunc: func(ctx context.Context, uID uuid.UUID) (*students.Student, error) {
				return &students.Student{ID: studentID, UserID: &userID, Status: "ACTIVE"}, nil
			},
			enrollClassTxFunc: func(ctx context.Context, sPlanID, sID, cID uuid.UUID, bypass bool) (*study_plans.EnrollmentDetail, error) {
				return nil, study_plans.ErrMaxCreditsExceeded
			},
		}
		svc := setupTestService(mock)
		principal := &auth.Principal{UserID: userID, Roles: []string{"mahasiswa"}}

		_, err := svc.EnrollClassInMyStudyPlan(context.Background(), principal, spID, classID)
		if err != study_plans.ErrMaxCreditsExceeded {
			t.Fatalf("expected ErrMaxCreditsExceeded, got: %v", err)
		}
	})

	t.Run("Enroll rejected if schedule conflict", func(t *testing.T) {
		conflictErr := appErrors.ErrConflict("Jadwal kelas bertabrakan dengan kelas lain")
		mock := &mockRepository{
			getStudentByUserIDFunc: func(ctx context.Context, uID uuid.UUID) (*students.Student, error) {
				return &students.Student{ID: studentID, UserID: &userID, Status: "ACTIVE"}, nil
			},
			enrollClassTxFunc: func(ctx context.Context, sPlanID, sID, cID uuid.UUID, bypass bool) (*study_plans.EnrollmentDetail, error) {
				return nil, conflictErr
			},
		}
		svc := setupTestService(mock)
		principal := &auth.Principal{UserID: userID, Roles: []string{"mahasiswa"}}

		_, err := svc.EnrollClassInMyStudyPlan(context.Background(), principal, spID, classID)
		if err != conflictErr {
			t.Fatalf("expected conflictErr, got: %v", err)
		}
	})

	t.Run("Drop enrollment success", func(t *testing.T) {
		mock := &mockRepository{
			getStudentByUserIDFunc: func(ctx context.Context, uID uuid.UUID) (*students.Student, error) {
				return &students.Student{ID: studentID, UserID: &userID, Status: "ACTIVE"}, nil
			},
			dropEnrollmentTxFunc: func(ctx context.Context, sPlanID, sID, eID uuid.UUID, bypass bool) error {
				return nil
			},
		}
		svc := setupTestService(mock)
		principal := &auth.Principal{UserID: userID, Roles: []string{"mahasiswa"}}

		if err := svc.DropEnrollmentFromMyStudyPlan(context.Background(), principal, spID, enrollmentID); err != nil {
			t.Fatalf("unexpected error dropping enrollment: %v", err)
		}
	})
}
