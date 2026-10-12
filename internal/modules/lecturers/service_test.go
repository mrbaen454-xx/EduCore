package lecturers_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"educore/internal/modules/lecturers"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
)

type mockLecturerRepo struct {
	studyPrograms map[uuid.UUID]bool // id -> isActive
	users         map[uuid.UUID]bool
	items         map[uuid.UUID]*lecturers.Lecturer
	byNIDN        map[string]*lecturers.Lecturer
	byEmail       map[string]*lecturers.Lecturer
}

func newMockLecturerRepo() *mockLecturerRepo {
	return &mockLecturerRepo{
		studyPrograms: make(map[uuid.UUID]bool),
		users:         make(map[uuid.UUID]bool),
		items:         make(map[uuid.UUID]*lecturers.Lecturer),
		byNIDN:        make(map[string]*lecturers.Lecturer),
		byEmail:       make(map[string]*lecturers.Lecturer),
	}
}

func (m *mockLecturerRepo) CheckStudyProgramActive(ctx context.Context, spID uuid.UUID) (bool, bool, error) {
	active, exists := m.studyPrograms[spID]
	return exists, active, nil
}

func (m *mockLecturerRepo) CheckUserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	return m.users[userID], nil
}

func (m *mockLecturerRepo) Create(ctx context.Context, l *lecturers.Lecturer) error {
	if _, exists := m.byNIDN[l.NIDN]; exists {
		return lecturers.ErrDuplicateNIDN
	}
	if _, exists := m.byEmail[l.Email]; exists {
		return lecturers.ErrDuplicateEmail
	}
	l.ID = uuid.New()
	l.StudyProgramCode = "TIF"
	l.StudyProgramName = "Teknik Informatika"
	l.FacultyName = "Fakultas Ilmu Komputer"
	m.items[l.ID] = l
	m.byNIDN[l.NIDN] = l
	m.byEmail[l.Email] = l
	return nil
}

func (m *mockLecturerRepo) GetByID(ctx context.Context, id uuid.UUID) (*lecturers.Lecturer, error) {
	l, exists := m.items[id]
	if !exists {
		return nil, lecturers.ErrNotFound
	}
	return l, nil
}

func (m *mockLecturerRepo) GetByNIDN(ctx context.Context, nidn string) (*lecturers.Lecturer, error) {
	l, exists := m.byNIDN[nidn]
	if !exists {
		return nil, lecturers.ErrNotFound
	}
	return l, nil
}

func (m *mockLecturerRepo) GetByEmail(ctx context.Context, email string) (*lecturers.Lecturer, error) {
	l, exists := m.byEmail[email]
	if !exists {
		return nil, lecturers.ErrNotFound
	}
	return l, nil
}

func (m *mockLecturerRepo) List(ctx context.Context, filter lecturers.LecturerFilter, p pagination.Params) ([]lecturers.Lecturer, int, error) {
	res := make([]lecturers.Lecturer, 0, len(m.items))
	for _, item := range m.items {
		if filter.StudyProgramID != nil && item.StudyProgramID != *filter.StudyProgramID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		res = append(res, *item)
	}
	return res, len(res), nil
}

func (m *mockLecturerRepo) Update(ctx context.Context, l *lecturers.Lecturer) error {
	if _, exists := m.items[l.ID]; !exists {
		return lecturers.ErrNotFound
	}
	m.items[l.ID] = l
	m.byNIDN[l.NIDN] = l
	m.byEmail[l.Email] = l
	return nil
}

func (m *mockLecturerRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	l, exists := m.items[id]
	if !exists {
		return lecturers.ErrNotFound
	}
	l.Status = status
	return nil
}

func TestLecturerService_Create(t *testing.T) {
	repo := newMockLecturerRepo()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := lecturers.NewService(repo, logger)
	ctx := context.Background()

	spID := uuid.New()
	repo.studyPrograms[spID] = true

	inactiveSpID := uuid.New()
	repo.studyPrograms[inactiveSpID] = false

	prefix := "Dr."
	suffix := "M.Kom."

	// 1. Valid creation
	req := lecturers.CreateLecturerRequest{
		StudyProgramID: spID,
		NIDN:           "0012345678",
		Name:           "Budi Raharjo",
		TitlePrefix:    &prefix,
		TitleSuffix:    &suffix,
		Email:          "budi@educore.ac.id",
	}
	res, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("expected create success, got: %v", err)
	}
	if res.NIDN != "0012345678" {
		t.Errorf("expected NIDN 0012345678, got %s", res.NIDN)
	}
	if res.FullName != "Dr. Budi Raharjo, M.Kom." {
		t.Errorf("expected full name 'Dr. Budi Raharjo, M.Kom.', got '%s'", res.FullName)
	}
	if res.Status != "ACTIVE" {
		t.Errorf("expected status ACTIVE, got %s", res.Status)
	}

	// 2. Duplicate NIDN
	reqDupNIDN := req
	reqDupNIDN.Email = "budi2@educore.ac.id"
	_, err = svc.Create(ctx, reqDupNIDN)
	if err == nil {
		t.Fatalf("expected error on duplicate NIDN, got nil")
	}
	appErr, ok := err.(*appErrors.AppError)
	if !ok || appErr.Code != appErrors.CodeConflict {
		t.Errorf("expected CONFLICT code, got %v", err)
	}

	// 3. Inactive study program
	reqInactiveSP := req
	reqInactiveSP.NIDN = "0088888888"
	reqInactiveSP.Email = "other@educore.ac.id"
	reqInactiveSP.StudyProgramID = inactiveSpID
	_, err = svc.Create(ctx, reqInactiveSP)
	if err == nil {
		t.Fatalf("expected error on inactive study program, got nil")
	}
}

func TestLecturerService_UpdateAndStatus(t *testing.T) {
	repo := newMockLecturerRepo()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := lecturers.NewService(repo, logger)
	ctx := context.Background()

	spID := uuid.New()
	repo.studyPrograms[spID] = true

	created, err := svc.Create(ctx, lecturers.CreateLecturerRequest{
		StudyProgramID: spID,
		NIDN:           "0011223344",
		Name:           "Ahmad Dahlan",
		Email:          "ahmad@educore.ac.id",
	})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Update phone and title
	phone := "08123456789"
	updated, err := svc.Update(ctx, created.ID, lecturers.UpdateLecturerRequest{Phone: &phone})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.Phone == nil || *updated.Phone != phone {
		t.Errorf("expected phone updated, got %v", updated.Phone)
	}

	// Update status to LEAVE
	res, err := svc.UpdateStatus(ctx, created.ID, "LEAVE")
	if err != nil {
		t.Fatalf("update status failed: %v", err)
	}
	if res.Status != "LEAVE" {
		t.Errorf("expected status LEAVE, got %s", res.Status)
	}
}
