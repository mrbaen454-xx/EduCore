package courses_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"educore/internal/modules/courses"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
)

type mockCourseRepo struct {
	items   map[uuid.UUID]*courses.Course
	byCode  map[string]*courses.Course
	prereqs map[uuid.UUID]map[uuid.UUID]bool // courseID -> prereqID -> true
}

func newMockCourseRepo() *mockCourseRepo {
	return &mockCourseRepo{
		items:   make(map[uuid.UUID]*courses.Course),
		byCode:  make(map[string]*courses.Course),
		prereqs: make(map[uuid.UUID]map[uuid.UUID]bool),
	}
}

func (m *mockCourseRepo) Create(ctx context.Context, c *courses.Course) error {
	if _, exists := m.byCode[c.Code]; exists {
		return courses.ErrDuplicateCode
	}
	c.ID = uuid.New()
	m.items[c.ID] = c
	m.byCode[c.Code] = c
	return nil
}

func (m *mockCourseRepo) GetByID(ctx context.Context, id uuid.UUID) (*courses.Course, error) {
	c, ok := m.items[id]
	if !ok {
		return nil, courses.ErrNotFound
	}
	return c, nil
}

func (m *mockCourseRepo) GetByCode(ctx context.Context, code string) (*courses.Course, error) {
	c, ok := m.byCode[code]
	if !ok {
		return nil, courses.ErrNotFound
	}
	return c, nil
}

func (m *mockCourseRepo) List(ctx context.Context, filter courses.ListCoursesFilter, p pagination.Params) ([]courses.Course, int, error) {
	var res []courses.Course
	for _, c := range m.items {
		res = append(res, *c)
	}
	return res, len(res), nil
}

func (m *mockCourseRepo) Update(ctx context.Context, c *courses.Course) error {
	if _, ok := m.items[c.ID]; !ok {
		return courses.ErrNotFound
	}
	m.items[c.ID] = c
	return nil
}

func (m *mockCourseRepo) UpdateStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	c, ok := m.items[id]
	if !ok {
		return courses.ErrNotFound
	}
	c.IsActive = isActive
	return nil
}

func (m *mockCourseRepo) AddPrerequisite(ctx context.Context, courseID, prereqID uuid.UUID) error {
	if courseID == prereqID {
		return courses.ErrSelfPrereq
	}
	if _, ok := m.items[prereqID]; !ok {
		return courses.ErrNotFound
	}
	if m.prereqs[courseID] == nil {
		m.prereqs[courseID] = make(map[uuid.UUID]bool)
	}
	if m.prereqs[courseID][prereqID] {
		return courses.ErrDuplicatePrereq
	}
	m.prereqs[courseID][prereqID] = true
	return nil
}

func (m *mockCourseRepo) RemovePrerequisite(ctx context.Context, courseID, prereqID uuid.UUID) error {
	if m.prereqs[courseID] != nil && m.prereqs[courseID][prereqID] {
		delete(m.prereqs[courseID], prereqID)
		return nil
	}
	return courses.ErrNotFound
}

func (m *mockCourseRepo) GetPrerequisites(ctx context.Context, courseID uuid.UUID) ([]courses.PrerequisiteDetail, error) {
	var list []courses.PrerequisiteDetail
	if m.prereqs[courseID] != nil {
		for pid := range m.prereqs[courseID] {
			if c, ok := m.items[pid]; ok {
				list = append(list, courses.PrerequisiteDetail{
					ID:      c.ID,
					Code:    c.Code,
					Name:    c.Name,
					Credits: c.Credits,
				})
			}
		}
	}
	return list, nil
}

func (m *mockCourseRepo) IsPrerequisite(ctx context.Context, courseID, prereqID uuid.UUID) (bool, error) {
	if m.prereqs[courseID] != nil && m.prereqs[courseID][prereqID] {
		return true, nil
	}
	return false, nil
}

func TestCourseService_Create(t *testing.T) {
	repo := newMockCourseRepo()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := courses.NewService(repo, logger)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		desc := "Dasar pemrograman"
		res, err := svc.Create(ctx, courses.CreateCourseRequest{
			Code:        "tif101",
			Name:        "Algoritma dan Pemrograman",
			Credits:     3,
			Description: &desc,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Code != "TIF101" {
			t.Errorf("expected code TIF101, got %s", res.Code)
		}
		if res.Credits != 3 {
			t.Errorf("expected 3 credits, got %d", res.Credits)
		}
	})

	t.Run("Duplicate Code", func(t *testing.T) {
		_, err := svc.Create(ctx, courses.CreateCourseRequest{
			Code:    "TIF101",
			Name:    "Alpro Lain",
			Credits: 4,
		})
		if err == nil {
			t.Fatalf("expected conflict error, got nil")
		}
		appErr, ok := err.(*appErrors.AppError)
		if !ok || appErr.Code != appErrors.CodeConflict {
			t.Errorf("expected CONFLICT code, got %v", err)
		}
	})

	t.Run("Invalid Credits", func(t *testing.T) {
		_, err := svc.Create(ctx, courses.CreateCourseRequest{
			Code:    "TIF102",
			Name:    "Mata Kuliah Nol SKS",
			Credits: 0,
		})
		if err == nil {
			t.Fatalf("expected error for 0 credits")
		}

		_, err2 := svc.Create(ctx, courses.CreateCourseRequest{
			Code:    "TIF103",
			Name:    "Mata Kuliah 10 SKS",
			Credits: 10,
		})
		if err2 == nil {
			t.Fatalf("expected error for >8 credits")
		}
	})
}

func TestCourseService_Prerequisites(t *testing.T) {
	repo := newMockCourseRepo()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := courses.NewService(repo, logger)
	ctx := context.Background()

	// Create Course A & Course B
	c1, _ := svc.Create(ctx, courses.CreateCourseRequest{Code: "CS101", Name: "Dasar Pemrograman", Credits: 3})
	c2, _ := svc.Create(ctx, courses.CreateCourseRequest{Code: "CS102", Name: "Struktur Data", Credits: 3})

	t.Run("Self Prerequisite Forbidden", func(t *testing.T) {
		err := svc.AddPrerequisite(ctx, c1.ID, courses.AddPrerequisiteRequest{
			PrerequisiteCourseID: c1.ID,
		})
		if err == nil {
			t.Fatalf("expected error for self prerequisite")
		}
	})

	t.Run("Add Prerequisite Success", func(t *testing.T) {
		err := svc.AddPrerequisite(ctx, c2.ID, courses.AddPrerequisiteRequest{
			PrerequisiteCourseID: c1.ID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		detail, err := svc.GetByID(ctx, c2.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(detail.Prerequisites) != 1 {
			t.Fatalf("expected 1 prerequisite, got %d", len(detail.Prerequisites))
		}
		if detail.Prerequisites[0].Code != "CS101" {
			t.Errorf("expected prerequisite CS101, got %s", detail.Prerequisites[0].Code)
		}
	})

	t.Run("Circular Prerequisite Forbidden", func(t *testing.T) {
		// Attempt to make CS102 prerequisite of CS101 (when CS101 is already prereq of CS102)
		err := svc.AddPrerequisite(ctx, c1.ID, courses.AddPrerequisiteRequest{
			PrerequisiteCourseID: c2.ID,
		})
		if err == nil {
			t.Fatalf("expected conflict error for circular prerequisite")
		}
		appErr, ok := err.(*appErrors.AppError)
		if !ok || appErr.Code != appErrors.CodeConflict {
			t.Errorf("expected CONFLICT code, got %v", err)
		}
	})
}
