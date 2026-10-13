package course_offerings_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"educore/internal/modules/course_offerings"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
)

type mockCourseOfferingRepo struct {
	offerings map[uuid.UUID]*course_offerings.CourseOffering
	byKey     map[string]*course_offerings.CourseOffering

	// entities state
	periods map[uuid.UUID]bool // id -> isActive
	courses map[uuid.UUID]bool // id -> isActive
	prodis  map[uuid.UUID]bool // id -> isActive
}

func newMockCourseOfferingRepo() *mockCourseOfferingRepo {
	return &mockCourseOfferingRepo{
		offerings: make(map[uuid.UUID]*course_offerings.CourseOffering),
		byKey:     make(map[string]*course_offerings.CourseOffering),
		periods:   make(map[uuid.UUID]bool),
		courses:   make(map[uuid.UUID]bool),
		prodis:    make(map[uuid.UUID]bool),
	}
}

func (m *mockCourseOfferingRepo) CheckForeignEntities(ctx context.Context, periodID, courseID, studyProgramID uuid.UUID, curriculumID *uuid.UUID) (bool, bool, bool, bool, error) {
	pActive, pOk := m.periods[periodID]
	if !pOk {
		return false, false, false, false, appErrors.ErrNotFound("Periode tidak ditemukan")
	}
	cActive, cOk := m.courses[courseID]
	if !cOk {
		return false, false, false, false, appErrors.ErrNotFound("Mata kuliah tidak ditemukan")
	}
	spActive, spOk := m.prodis[studyProgramID]
	if !spOk {
		return false, false, false, false, appErrors.ErrNotFound("Prodi tidak ditemukan")
	}
	return pActive, cActive, spActive, true, nil
}

func (m *mockCourseOfferingRepo) Create(ctx context.Context, co *course_offerings.CourseOffering) error {
	key := co.AcademicPeriodID.String() + ":" + co.CourseID.String() + ":" + co.StudyProgramID.String()
	if _, exists := m.byKey[key]; exists {
		return course_offerings.ErrDuplicateOffering
	}
	co.ID = uuid.New()
	co.CreatedAt = time.Now()
	co.UpdatedAt = time.Now()
	m.offerings[co.ID] = co
	m.byKey[key] = co
	return nil
}

func (m *mockCourseOfferingRepo) GetByID(ctx context.Context, id uuid.UUID) (*course_offerings.CourseOfferingDetail, error) {
	co, exists := m.offerings[id]
	if !exists {
		return nil, course_offerings.ErrNotFound
	}
	return &course_offerings.CourseOfferingDetail{
		CourseOffering:     *co,
		AcademicPeriodCode: "20241",
		CourseCode:         "CS101",
		CourseName:         "Alpro",
		CourseCredits:      3,
		StudyProgramCode:   "TIF",
		StudyProgramName:   "Teknik Informatika",
	}, nil
}

func (m *mockCourseOfferingRepo) List(ctx context.Context, filter course_offerings.CourseOfferingFilter, p pagination.Params) ([]course_offerings.CourseOfferingDetail, int, error) {
	var list []course_offerings.CourseOfferingDetail
	for _, co := range m.offerings {
		list = append(list, course_offerings.CourseOfferingDetail{
			CourseOffering: *co,
		})
	}
	return list, len(list), nil
}

func (m *mockCourseOfferingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	co, exists := m.offerings[id]
	if !exists {
		return course_offerings.ErrNotFound
	}
	co.Status = status
	co.UpdatedAt = time.Now()
	return nil
}

func TestCourseOfferingService(t *testing.T) {
	repo := newMockCourseOfferingRepo()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := course_offerings.NewService(repo, logger)
	ctx := context.Background()

	periodID := uuid.New()
	repo.periods[periodID] = true

	activeCourseID := uuid.New()
	repo.courses[activeCourseID] = true

	inactiveCourseID := uuid.New()
	repo.courses[inactiveCourseID] = false

	activeProdiID := uuid.New()
	repo.prodis[activeProdiID] = true

	inactiveProdiID := uuid.New()
	repo.prodis[inactiveProdiID] = false

	t.Run("Create Success", func(t *testing.T) {
		res, err := svc.Create(ctx, course_offerings.CreateCourseOfferingRequest{
			AcademicPeriodID: periodID,
			CourseID:         activeCourseID,
			StudyProgramID:   activeProdiID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != "PLANNED" {
			t.Errorf("expected PLANNED status, got %s", res.Status)
		}
	})

	t.Run("Reject Duplicate Offering", func(t *testing.T) {
		_, err := svc.Create(ctx, course_offerings.CreateCourseOfferingRequest{
			AcademicPeriodID: periodID,
			CourseID:         activeCourseID,
			StudyProgramID:   activeProdiID,
		})
		if err == nil {
			t.Fatalf("expected conflict error for duplicate offering")
		}
		appErr, ok := err.(*appErrors.AppError)
		if !ok || appErr.Code != appErrors.CodeConflict {
			t.Errorf("expected CONFLICT code, got %v", err)
		}
	})

	t.Run("Reject Inactive Course", func(t *testing.T) {
		_, err := svc.Create(ctx, course_offerings.CreateCourseOfferingRequest{
			AcademicPeriodID: periodID,
			CourseID:         inactiveCourseID,
			StudyProgramID:   activeProdiID,
		})
		if err == nil {
			t.Fatalf("expected conflict error for inactive course")
		}
	})

	t.Run("Reject Inactive Study Program", func(t *testing.T) {
		anotherCourseID := uuid.New()
		repo.courses[anotherCourseID] = true

		_, err := svc.Create(ctx, course_offerings.CreateCourseOfferingRequest{
			AcademicPeriodID: periodID,
			CourseID:         anotherCourseID,
			StudyProgramID:   inactiveProdiID,
		})
		if err == nil {
			t.Fatalf("expected conflict error for inactive prodi")
		}
	})
}
