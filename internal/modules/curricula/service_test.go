package curricula_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"educore/internal/modules/curricula"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
)

type mockCurriculumRepo struct {
	studyPrograms map[uuid.UUID]bool // id -> isActive
	courses       map[uuid.UUID]struct {
		credits  int
		code     string
		name     string
		isActive bool
	}
	items        map[uuid.UUID]*curricula.Curriculum
	byCode       map[string]*curricula.Curriculum
	curriCourses map[uuid.UUID]map[uuid.UUID]*curricula.CurriculumCourse
}

func newMockCurriculumRepo() *mockCurriculumRepo {
	return &mockCurriculumRepo{
		studyPrograms: make(map[uuid.UUID]bool),
		courses: make(map[uuid.UUID]struct {
			credits  int
			code     string
			name     string
			isActive bool
		}),
		items:        make(map[uuid.UUID]*curricula.Curriculum),
		byCode:       make(map[string]*curricula.Curriculum),
		curriCourses: make(map[uuid.UUID]map[uuid.UUID]*curricula.CurriculumCourse),
	}
}

func (m *mockCurriculumRepo) Create(ctx context.Context, c *curricula.Curriculum) error {
	if _, exists := m.byCode[c.Code]; exists {
		return curricula.ErrDuplicateCode
	}
	c.ID = uuid.New()
	m.items[c.ID] = c
	m.byCode[c.Code] = c
	return nil
}

func (m *mockCurriculumRepo) GetByID(ctx context.Context, id uuid.UUID) (*curricula.Curriculum, error) {
	c, ok := m.items[id]
	if !ok {
		return nil, curricula.ErrNotFound
	}
	return c, nil
}

func (m *mockCurriculumRepo) GetByCode(ctx context.Context, code string) (*curricula.Curriculum, error) {
	c, ok := m.byCode[code]
	if !ok {
		return nil, curricula.ErrNotFound
	}
	return c, nil
}

func (m *mockCurriculumRepo) List(ctx context.Context, filter curricula.ListCurriculaFilter, p pagination.Params) ([]curricula.Curriculum, int, error) {
	var res []curricula.Curriculum
	for _, c := range m.items {
		res = append(res, *c)
	}
	return res, len(res), nil
}

func (m *mockCurriculumRepo) Update(ctx context.Context, c *curricula.Curriculum) error {
	if _, ok := m.items[c.ID]; !ok {
		return curricula.ErrNotFound
	}
	m.items[c.ID] = c
	return nil
}

func (m *mockCurriculumRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	c, ok := m.items[id]
	if !ok {
		return curricula.ErrNotFound
	}
	c.Status = status
	return nil
}

func (m *mockCurriculumRepo) GetStats(ctx context.Context, curriculumID uuid.UUID) (*curricula.CurriculumStats, error) {
	stats := &curricula.CurriculumStats{}
	coursesMap := m.curriCourses[curriculumID]
	if coursesMap != nil {
		for _, cc := range coursesMap {
			stats.TotalCourses++
			courseInfo := m.courses[cc.CourseID]
			stats.TotalCredits += courseInfo.credits
			if cc.Category == curricula.CategoryRequired {
				stats.RequiredCredits += courseInfo.credits
			} else {
				stats.ElectiveCredits += courseInfo.credits
			}
		}
	}
	return stats, nil
}

func (m *mockCurriculumRepo) CheckStudyProgramActive(ctx context.Context, studyProgramID uuid.UUID) (bool, error) {
	isActive, ok := m.studyPrograms[studyProgramID]
	if !ok {
		return false, curricula.ErrStudyProgramNotFound
	}
	return isActive, nil
}

func (m *mockCurriculumRepo) AddCourse(ctx context.Context, cc *curricula.CurriculumCourse) error {
	if m.curriCourses[cc.CurriculumID] == nil {
		m.curriCourses[cc.CurriculumID] = make(map[uuid.UUID]*curricula.CurriculumCourse)
	}
	if _, exists := m.curriCourses[cc.CurriculumID][cc.CourseID]; exists {
		return curricula.ErrDuplicateCourse
	}
	cc.ID = uuid.New()
	m.curriCourses[cc.CurriculumID][cc.CourseID] = cc
	return nil
}

func (m *mockCurriculumRepo) GetCourse(ctx context.Context, curriculumID, courseID uuid.UUID) (*curricula.CurriculumCourseDetail, error) {
	coursesMap := m.curriCourses[curriculumID]
	if coursesMap == nil {
		return nil, curricula.ErrCourseNotFound
	}
	cc, ok := coursesMap[courseID]
	if !ok {
		return nil, curricula.ErrCourseNotFound
	}
	info := m.courses[courseID]
	return &curricula.CurriculumCourseDetail{
		ID:           cc.ID,
		CurriculumID: cc.CurriculumID,
		CourseID:     cc.CourseID,
		CourseCode:   info.code,
		CourseName:   info.name,
		Credits:      info.credits,
		Semester:     cc.Semester,
		Category:     cc.Category,
	}, nil
}

func (m *mockCurriculumRepo) ListCourses(ctx context.Context, curriculumID uuid.UUID, filter curricula.ListCurriculumCoursesFilter, p pagination.Params) ([]curricula.CurriculumCourseDetail, int, error) {
	var list []curricula.CurriculumCourseDetail
	coursesMap := m.curriCourses[curriculumID]
	if coursesMap != nil {
		for _, cc := range coursesMap {
			info := m.courses[cc.CourseID]
			list = append(list, curricula.CurriculumCourseDetail{
				ID:           cc.ID,
				CurriculumID: cc.CurriculumID,
				CourseID:     cc.CourseID,
				CourseCode:   info.code,
				CourseName:   info.name,
				Credits:      info.credits,
				Semester:     cc.Semester,
				Category:     cc.Category,
			})
		}
	}
	return list, len(list), nil
}

func (m *mockCurriculumRepo) UpdateCourse(ctx context.Context, curriculumID, courseID uuid.UUID, semester *int, category *string) error {
	coursesMap := m.curriCourses[curriculumID]
	if coursesMap == nil || coursesMap[courseID] == nil {
		return curricula.ErrCourseNotFound
	}
	cc := coursesMap[courseID]
	if semester != nil {
		cc.Semester = *semester
	}
	if category != nil {
		cc.Category = *category
	}
	return nil
}

func (m *mockCurriculumRepo) RemoveCourse(ctx context.Context, curriculumID, courseID uuid.UUID) error {
	coursesMap := m.curriCourses[curriculumID]
	if coursesMap == nil || coursesMap[courseID] == nil {
		return curricula.ErrCourseNotFound
	}
	delete(coursesMap, courseID)
	return nil
}

func (m *mockCurriculumRepo) CheckCourseActive(ctx context.Context, courseID uuid.UUID) (bool, error) {
	c, ok := m.courses[courseID]
	if !ok {
		return false, curricula.ErrCourseNotFound
	}
	return c.isActive, nil
}

func TestCurriculumService_Create(t *testing.T) {
	repo := newMockCurriculumRepo()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := curricula.NewService(repo, logger)
	ctx := context.Background()

	activeProdiID := uuid.New()
	inactiveProdiID := uuid.New()
	repo.studyPrograms[activeProdiID] = true
	repo.studyPrograms[inactiveProdiID] = false

	t.Run("Success", func(t *testing.T) {
		res, err := svc.Create(ctx, curricula.CreateCurriculumRequest{
			StudyProgramID: activeProdiID,
			Code:           "kur-tif-2024",
			Name:           "Kurikulum Teknik Informatika 2024",
			Year:           2024,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Code != "KUR-TIF-2024" {
			t.Errorf("expected code KUR-TIF-2024, got %s", res.Code)
		}
		if res.Status != curricula.StatusDraft {
			t.Errorf("expected default status DRAFT, got %s", res.Status)
		}
	})

	t.Run("Reject Inactive Study Program", func(t *testing.T) {
		_, err := svc.Create(ctx, curricula.CreateCurriculumRequest{
			StudyProgramID: inactiveProdiID,
			Code:           "KUR-OLD",
			Name:           "Kurikulum Prodi Nonaktif",
			Year:           2024,
		})
		if err == nil {
			t.Fatalf("expected conflict error, got nil")
		}
		appErr, ok := err.(*appErrors.AppError)
		if !ok || appErr.Code != appErrors.CodeConflict {
			t.Errorf("expected CONFLICT code, got %v", err)
		}
	})
}

func TestCurriculumService_CourseAssociation(t *testing.T) {
	repo := newMockCurriculumRepo()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := curricula.NewService(repo, logger)
	ctx := context.Background()

	spID := uuid.New()
	repo.studyPrograms[spID] = true

	curr, _ := svc.Create(ctx, curricula.CreateCurriculumRequest{
		StudyProgramID: spID,
		Code:           "KUR-2024",
		Name:           "Kurikulum 2024",
		Year:           2024,
	})

	courseActiveID := uuid.New()
	courseInactiveID := uuid.New()
	repo.courses[courseActiveID] = struct {
		credits  int
		code     string
		name     string
		isActive bool
	}{credits: 3, code: "CS101", name: "Alpro", isActive: true}

	repo.courses[courseInactiveID] = struct {
		credits  int
		code     string
		name     string
		isActive bool
	}{credits: 2, code: "CS999", name: "Nonaktif", isActive: false}

	t.Run("Add Active Course Success", func(t *testing.T) {
		detail, err := svc.AddCourse(ctx, curr.ID, curricula.AddCurriculumCourseRequest{
			CourseID: courseActiveID,
			Semester: 1,
			Category: "REQUIRED",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detail.CourseCode != "CS101" {
			t.Errorf("expected course CS101, got %s", detail.CourseCode)
		}

		// Verify stats updated
		currDetail, err := svc.GetByID(ctx, curr.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if currDetail.Stats.TotalCredits != 3 {
			t.Errorf("expected 3 total credits, got %d", currDetail.Stats.TotalCredits)
		}
	})

	t.Run("Reject Duplicate Course in Curriculum", func(t *testing.T) {
		_, err := svc.AddCourse(ctx, curr.ID, curricula.AddCurriculumCourseRequest{
			CourseID: courseActiveID,
			Semester: 2,
			Category: "ELECTIVE",
		})
		if err == nil {
			t.Fatalf("expected conflict error, got nil")
		}
		appErr, ok := err.(*appErrors.AppError)
		if !ok || appErr.Code != appErrors.CodeConflict {
			t.Errorf("expected CONFLICT code, got %v", err)
		}
	})

	t.Run("Reject Inactive Course", func(t *testing.T) {
		_, err := svc.AddCourse(ctx, curr.ID, curricula.AddCurriculumCourseRequest{
			CourseID: courseInactiveID,
			Semester: 1,
			Category: "ELECTIVE",
		})
		if err == nil {
			t.Fatalf("expected conflict error for inactive course")
		}
	})
}
