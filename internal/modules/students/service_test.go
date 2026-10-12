package students_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"educore/internal/modules/students"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
)

type mockStudentRepo struct {
	studyPrograms map[uuid.UUID]bool // id -> isActive
	users         map[uuid.UUID]bool
	items         map[uuid.UUID]*students.Student
	byNIM         map[string]*students.Student
	byEmail       map[string]*students.Student
}

func newMockStudentRepo() *mockStudentRepo {
	return &mockStudentRepo{
		studyPrograms: make(map[uuid.UUID]bool),
		users:         make(map[uuid.UUID]bool),
		items:         make(map[uuid.UUID]*students.Student),
		byNIM:         make(map[string]*students.Student),
		byEmail:       make(map[string]*students.Student),
	}
}

func (m *mockStudentRepo) CheckStudyProgramActive(ctx context.Context, spID uuid.UUID) (bool, bool, error) {
	active, exists := m.studyPrograms[spID]
	return exists, active, nil
}

func (m *mockStudentRepo) CheckUserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	return m.users[userID], nil
}

func (m *mockStudentRepo) Create(ctx context.Context, s *students.Student) error {
	if _, exists := m.byNIM[s.NIM]; exists {
		return students.ErrDuplicateNIM
	}
	if _, exists := m.byEmail[s.Email]; exists {
		return students.ErrDuplicateEmail
	}
	s.ID = uuid.New()
	s.StudyProgramCode = "TIF"
	s.StudyProgramName = "Teknik Informatika"
	s.FacultyName = "Fakultas Ilmu Komputer"
	m.items[s.ID] = s
	m.byNIM[s.NIM] = s
	m.byEmail[s.Email] = s
	return nil
}

func (m *mockStudentRepo) GetByID(ctx context.Context, id uuid.UUID) (*students.Student, error) {
	s, exists := m.items[id]
	if !exists {
		return nil, students.ErrNotFound
	}
	return s, nil
}

func (m *mockStudentRepo) GetByNIM(ctx context.Context, nim string) (*students.Student, error) {
	s, exists := m.byNIM[nim]
	if !exists {
		return nil, students.ErrNotFound
	}
	return s, nil
}

func (m *mockStudentRepo) GetByEmail(ctx context.Context, email string) (*students.Student, error) {
	s, exists := m.byEmail[email]
	if !exists {
		return nil, students.ErrNotFound
	}
	return s, nil
}

func (m *mockStudentRepo) List(ctx context.Context, filter students.StudentFilter, p pagination.Params) ([]students.Student, int, error) {
	res := make([]students.Student, 0, len(m.items))
	for _, item := range m.items {
		if filter.StudyProgramID != nil && item.StudyProgramID != *filter.StudyProgramID {
			continue
		}
		if filter.AdmissionYear != nil && item.AdmissionYear != *filter.AdmissionYear {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		res = append(res, *item)
	}
	return res, len(res), nil
}

func (m *mockStudentRepo) Update(ctx context.Context, s *students.Student) error {
	if _, exists := m.items[s.ID]; !exists {
		return students.ErrNotFound
	}
	m.items[s.ID] = s
	m.byNIM[s.NIM] = s
	m.byEmail[s.Email] = s
	return nil
}

func (m *mockStudentRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	s, exists := m.items[id]
	if !exists {
		return students.ErrNotFound
	}
	s.Status = status
	return nil
}

func TestStudentService_Create(t *testing.T) {
	repo := newMockStudentRepo()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := students.NewService(repo, logger)
	ctx := context.Background()

	spID := uuid.New()
	repo.studyPrograms[spID] = true

	gender := "MALE"
	birthDate := "2002-05-15"

	// 1. Valid creation
	req := students.CreateStudentRequest{
		StudyProgramID: spID,
		NIM:            "2024001001",
		Name:           "Reza Rahadian",
		Email:          "reza@student.educore.ac.id",
		Gender:         &gender,
		BirthDate:      &birthDate,
		AdmissionYear:  2024,
	}
	res, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("expected create success, got: %v", err)
	}
	if res.NIM != "2024001001" {
		t.Errorf("expected NIM 2024001001, got %s", res.NIM)
	}
	if res.Status != "ACTIVE" {
		t.Errorf("expected status ACTIVE, got %s", res.Status)
	}

	// 2. Duplicate NIM
	reqDupNIM := req
	reqDupNIM.Email = "reza2@student.educore.ac.id"
	_, err = svc.Create(ctx, reqDupNIM)
	if err == nil {
		t.Fatalf("expected error on duplicate NIM, got nil")
	}
	appErr, ok := err.(*appErrors.AppError)
	if !ok || appErr.Code != appErrors.CodeConflict {
		t.Errorf("expected CONFLICT code, got %v", err)
	}

	// 3. Invalid BirthDate
	reqBadDate := req
	reqBadDate.NIM = "2024001002"
	reqBadDate.Email = "reza3@student.educore.ac.id"
	badDate := "15-05-2002"
	reqBadDate.BirthDate = &badDate
	_, err = svc.Create(ctx, reqBadDate)
	if err == nil {
		t.Fatalf("expected error on invalid birth date, got nil")
	}

	// 4. Invalid AdmissionYear
	reqBadYear := req
	reqBadYear.NIM = "2024001003"
	reqBadYear.Email = "reza4@student.educore.ac.id"
	reqBadYear.AdmissionYear = 1960
	_, err = svc.Create(ctx, reqBadYear)
	if err == nil {
		t.Fatalf("expected error on invalid admission year, got nil")
	}
}

func TestStudentService_UpdateAndStatus(t *testing.T) {
	repo := newMockStudentRepo()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := students.NewService(repo, logger)
	ctx := context.Background()

	spID := uuid.New()
	repo.studyPrograms[spID] = true

	created, err := svc.Create(ctx, students.CreateStudentRequest{
		StudyProgramID: spID,
		NIM:            "2024002001",
		Name:           "Siti Aisyah",
		Email:          "siti@student.educore.ac.id",
		AdmissionYear:  2024,
	})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Update phone
	phone := "08987654321"
	updated, err := svc.Update(ctx, created.ID, students.UpdateStudentRequest{Phone: &phone})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.Phone == nil || *updated.Phone != phone {
		t.Errorf("expected phone updated, got %v", updated.Phone)
	}

	// Update status to GRADUATED
	res, err := svc.UpdateStatus(ctx, created.ID, "GRADUATED")
	if err != nil {
		t.Fatalf("update status failed: %v", err)
	}
	if res.Status != "GRADUATED" {
		t.Errorf("expected status GRADUATED, got %s", res.Status)
	}
}
