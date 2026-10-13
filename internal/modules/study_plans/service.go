package study_plans

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"educore/internal/modules/auth"
	"educore/internal/modules/students"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
)

type Service interface {
	// Student-facing workflows
	GetStudentProfile(ctx context.Context, principal *auth.Principal) (*students.Student, error)
	GetOrCreateMyStudyPlan(ctx context.Context, principal *auth.Principal, periodID uuid.UUID) (*StudyPlanDetail, bool, error)
	GetMyStudyPlanDetail(ctx context.Context, principal *auth.Principal, studyPlanID uuid.UUID) (*StudyPlanDetail, error)
	ListMyStudyPlans(ctx context.Context, principal *auth.Principal, filter StudyPlanFilter, p pagination.Params) ([]StudyPlanDetail, int, error)
	EnrollClassInMyStudyPlan(ctx context.Context, principal *auth.Principal, studyPlanID, classID uuid.UUID) (*EnrollmentDetail, error)
	DropEnrollmentFromMyStudyPlan(ctx context.Context, principal *auth.Principal, studyPlanID, enrollmentID uuid.UUID) error

	// Administrative workflows
	CreateStudyPlanForAdmin(ctx context.Context, req CreateStudyPlanRequest) (*StudyPlanDetail, bool, error)
	GetStudyPlanDetailForAdmin(ctx context.Context, studyPlanID uuid.UUID) (*StudyPlanDetail, error)
	ListStudyPlansForAdmin(ctx context.Context, filter StudyPlanFilter, p pagination.Params) ([]StudyPlanDetail, int, error)
	EnrollClassForAdmin(ctx context.Context, studyPlanID, classID uuid.UUID) (*EnrollmentDetail, error)
	DropEnrollmentForAdmin(ctx context.Context, studyPlanID, enrollmentID uuid.UUID) error

	// Common query workflows
	ListAvailableClasses(ctx context.Context, filter AvailableClassFilter, p pagination.Params) ([]AvailableClass, int, error)
	GetActiveAcademicPeriodID(ctx context.Context) (uuid.UUID, error)
}

type service struct {
	repo   Repository
	logger *slog.Logger
}

func NewService(repo Repository, logger *slog.Logger) Service {
	return &service{
		repo:   repo,
		logger: logger,
	}
}

func (s *service) GetStudentProfile(ctx context.Context, principal *auth.Principal) (*students.Student, error) {
	if principal == nil {
		return nil, appErrors.ErrUnauthorized("Autentikasi diperlukan")
	}

	student, err := s.repo.GetStudentByUserID(ctx, principal.UserID)
	if err != nil {
		return nil, err
	}

	if student.Status != "ACTIVE" {
		return nil, appErrors.ErrConflict(fmt.Sprintf("Mahasiswa berstatus %s tidak memenuhi syarat untuk pengisian KRS", student.Status))
	}

	return student, nil
}

func (s *service) GetOrCreateMyStudyPlan(ctx context.Context, principal *auth.Principal, periodID uuid.UUID) (*StudyPlanDetail, bool, error) {
	student, err := s.GetStudentProfile(ctx, principal)
	if err != nil {
		return nil, false, err
	}

	// Verify academic period
	exists, active, err := s.repo.CheckAcademicPeriodActive(ctx, periodID)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, ErrAcademicPeriodNotFound
	}
	if !active {
		return nil, false, ErrAcademicPeriodNotActive
	}

	// Check if already exists
	existing, err := s.repo.GetStudyPlanByStudentAndPeriod(ctx, student.ID, periodID)
	if err == nil && existing != nil {
		return existing, false, nil
	}

	// Create new study plan
	now := time.Now()
	newSP := &StudyPlan{
		StudentID:        student.ID,
		AcademicPeriodID: periodID,
		TotalCredits:     0,
		Status:           "ACTIVE",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.repo.CreateStudyPlan(ctx, newSP); err != nil {
		// If duplicate from concurrent create, fetch existing
		if err == ErrDuplicateStudyPlan {
			existing, errFetch := s.repo.GetStudyPlanByStudentAndPeriod(ctx, student.ID, periodID)
			if errFetch == nil {
				return existing, false, nil
			}
		}
		return nil, false, err
	}

	created, err := s.repo.GetStudyPlanByID(ctx, newSP.ID)
	if err != nil {
		return nil, false, err
	}
	return created, true, nil
}

func (s *service) GetMyStudyPlanDetail(ctx context.Context, principal *auth.Principal, studyPlanID uuid.UUID) (*StudyPlanDetail, error) {
	student, err := s.GetStudentProfile(ctx, principal)
	if err != nil {
		return nil, err
	}

	sp, err := s.repo.GetStudyPlanByID(ctx, studyPlanID)
	if err != nil {
		return nil, err
	}

	if sp.StudentID != student.ID {
		return nil, ErrUnauthorizedStudyPlan
	}

	return sp, nil
}

func (s *service) ListMyStudyPlans(ctx context.Context, principal *auth.Principal, filter StudyPlanFilter, p pagination.Params) ([]StudyPlanDetail, int, error) {
	student, err := s.GetStudentProfile(ctx, principal)
	if err != nil {
		return nil, 0, err
	}

	filter.StudentID = &student.ID
	return s.repo.ListStudyPlans(ctx, filter, p)
}

func (s *service) EnrollClassInMyStudyPlan(ctx context.Context, principal *auth.Principal, studyPlanID, classID uuid.UUID) (*EnrollmentDetail, error) {
	student, err := s.GetStudentProfile(ctx, principal)
	if err != nil {
		return nil, err
	}

	return s.repo.EnrollClassTx(ctx, studyPlanID, student.ID, classID, false)
}

func (s *service) DropEnrollmentFromMyStudyPlan(ctx context.Context, principal *auth.Principal, studyPlanID, enrollmentID uuid.UUID) error {
	student, err := s.GetStudentProfile(ctx, principal)
	if err != nil {
		return err
	}

	return s.repo.DropEnrollmentTx(ctx, studyPlanID, student.ID, enrollmentID, false)
}

func (s *service) CreateStudyPlanForAdmin(ctx context.Context, req CreateStudyPlanRequest) (*StudyPlanDetail, bool, error) {
	if req.StudentID == nil {
		return nil, false, appErrors.ErrBadRequest("student_id wajib disertakan untuk pembuatan KRS oleh admin")
	}

	student, err := s.repo.GetStudentByID(ctx, *req.StudentID)
	if err != nil {
		return nil, false, err
	}

	if student.Status != "ACTIVE" {
		return nil, false, appErrors.ErrConflict(fmt.Sprintf("Mahasiswa berstatus %s tidak memenuhi syarat untuk pengisian KRS", student.Status))
	}

	exists, active, err := s.repo.CheckAcademicPeriodActive(ctx, req.AcademicPeriodID)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, ErrAcademicPeriodNotFound
	}
	if !active {
		return nil, false, ErrAcademicPeriodNotActive
	}

	existing, err := s.repo.GetStudyPlanByStudentAndPeriod(ctx, *req.StudentID, req.AcademicPeriodID)
	if err == nil && existing != nil {
		return existing, false, nil
	}

	now := time.Now()
	newSP := &StudyPlan{
		StudentID:        *req.StudentID,
		AcademicPeriodID: req.AcademicPeriodID,
		TotalCredits:     0,
		Status:           "ACTIVE",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.repo.CreateStudyPlan(ctx, newSP); err != nil {
		if err == ErrDuplicateStudyPlan {
			existing, errFetch := s.repo.GetStudyPlanByStudentAndPeriod(ctx, *req.StudentID, req.AcademicPeriodID)
			if errFetch == nil {
				return existing, false, nil
			}
		}
		return nil, false, err
	}

	created, err := s.repo.GetStudyPlanByID(ctx, newSP.ID)
	if err != nil {
		return nil, false, err
	}
	return created, true, nil
}

func (s *service) GetStudyPlanDetailForAdmin(ctx context.Context, studyPlanID uuid.UUID) (*StudyPlanDetail, error) {
	return s.repo.GetStudyPlanByID(ctx, studyPlanID)
}

func (s *service) ListStudyPlansForAdmin(ctx context.Context, filter StudyPlanFilter, p pagination.Params) ([]StudyPlanDetail, int, error) {
	return s.repo.ListStudyPlans(ctx, filter, p)
}

func (s *service) EnrollClassForAdmin(ctx context.Context, studyPlanID, classID uuid.UUID) (*EnrollmentDetail, error) {
	return s.repo.EnrollClassTx(ctx, studyPlanID, uuid.Nil, classID, true)
}

func (s *service) DropEnrollmentForAdmin(ctx context.Context, studyPlanID, enrollmentID uuid.UUID) error {
	return s.repo.DropEnrollmentTx(ctx, studyPlanID, uuid.Nil, enrollmentID, true)
}

func (s *service) ListAvailableClasses(ctx context.Context, filter AvailableClassFilter, p pagination.Params) ([]AvailableClass, int, error) {
	return s.repo.ListAvailableClasses(ctx, filter, p)
}

func (s *service) GetActiveAcademicPeriodID(ctx context.Context) (uuid.UUID, error) {
	return s.repo.GetActiveAcademicPeriodID(ctx)
}
