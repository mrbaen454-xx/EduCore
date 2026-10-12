package study_programs_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"educore/internal/modules/study_programs"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
)

type mockSPRepo struct {
	faculties map[uuid.UUID]bool // id -> isActive
	items     map[uuid.UUID]*study_programs.StudyProgram
	byCode    map[string]*study_programs.StudyProgram
}

func newMockSPRepo() *mockSPRepo {
	return &mockSPRepo{
		faculties: make(map[uuid.UUID]bool),
		items:     make(map[uuid.UUID]*study_programs.StudyProgram),
		byCode:    make(map[string]*study_programs.StudyProgram),
	}
}

func (m *mockSPRepo) CheckFacultyActive(ctx context.Context, facultyID uuid.UUID) (bool, bool, error) {
	active, exists := m.faculties[facultyID]
	return exists, active, nil
}

func (m *mockSPRepo) Create(ctx context.Context, sp *study_programs.StudyProgram) error {
	if _, exists := m.byCode[sp.Code]; exists {
		return study_programs.ErrDuplicateCode
	}
	sp.ID = uuid.New()
	sp.FacultyCode = "FIK"
	sp.FacultyName = "Fakultas Ilmu Komputer"
	m.items[sp.ID] = sp
	m.byCode[sp.Code] = sp
	return nil
}

func (m *mockSPRepo) GetByID(ctx context.Context, id uuid.UUID) (*study_programs.StudyProgram, error) {
	sp, exists := m.items[id]
	if !exists {
		return nil, study_programs.ErrNotFound
	}
	return sp, nil
}

func (m *mockSPRepo) GetByCode(ctx context.Context, code string) (*study_programs.StudyProgram, error) {
	sp, exists := m.byCode[code]
	if !exists {
		return nil, study_programs.ErrNotFound
	}
	return sp, nil
}

func (m *mockSPRepo) List(ctx context.Context, filter study_programs.StudyProgramFilter, p pagination.Params) ([]study_programs.StudyProgram, int, error) {
	res := make([]study_programs.StudyProgram, 0, len(m.items))
	for _, item := range m.items {
		if filter.FacultyID != nil && item.FacultyID != *filter.FacultyID {
			continue
		}
		if filter.Level != "" && item.Level != filter.Level {
			continue
		}
		if filter.IsActive != nil && item.IsActive != *filter.IsActive {
			continue
		}
		res = append(res, *item)
	}
	return res, len(res), nil
}

func (m *mockSPRepo) Update(ctx context.Context, sp *study_programs.StudyProgram) error {
	if _, exists := m.items[sp.ID]; !exists {
		return study_programs.ErrNotFound
	}
	m.items[sp.ID] = sp
	m.byCode[sp.Code] = sp
	return nil
}

func (m *mockSPRepo) UpdateStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	sp, exists := m.items[id]
	if !exists {
		return study_programs.ErrNotFound
	}
	sp.IsActive = isActive
	return nil
}

func TestStudyProgramService_Create(t *testing.T) {
	repo := newMockSPRepo()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := study_programs.NewService(repo, logger)
	ctx := context.Background()

	activeFacultyID := uuid.New()
	repo.faculties[activeFacultyID] = true

	inactiveFacultyID := uuid.New()
	repo.faculties[inactiveFacultyID] = false

	// 1. Valid creation
	req := study_programs.CreateStudyProgramRequest{
		FacultyID: activeFacultyID,
		Code:      "tif",
		Name:      "Teknik Informatika",
		Level:     "S1",
	}
	res, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("expected create success, got: %v", err)
	}
	if res.Code != "TIF" {
		t.Errorf("expected code TIF, got %s", res.Code)
	}
	if res.Level != "S1" {
		t.Errorf("expected level S1, got %s", res.Level)
	}

	// 2. Inactive faculty
	reqInactive := study_programs.CreateStudyProgramRequest{
		FacultyID: inactiveFacultyID,
		Code:      "si",
		Name:      "Sistem Informasi",
		Level:     "S1",
	}
	_, err = svc.Create(ctx, reqInactive)
	if err == nil {
		t.Fatalf("expected error on inactive faculty, got nil")
	}
	appErr, ok := err.(*appErrors.AppError)
	if !ok || appErr.Code != appErrors.CodeConflict {
		t.Errorf("expected CONFLICT code, got %v", err)
	}

	// 3. Non-existent faculty
	reqUnknown := study_programs.CreateStudyProgramRequest{
		FacultyID: uuid.New(),
		Code:      "mi",
		Name:      "Manajemen Informatika",
		Level:     "D3",
	}
	_, err = svc.Create(ctx, reqUnknown)
	if err == nil {
		t.Fatalf("expected error on non-existent faculty, got nil")
	}
	appErr, ok = err.(*appErrors.AppError)
	if !ok || appErr.Code != appErrors.CodeResourceNotFound {
		t.Errorf("expected RESOURCE_NOT_FOUND code, got %v", err)
	}

	// 4. Invalid level
	reqBadLevel := study_programs.CreateStudyProgramRequest{
		FacultyID: activeFacultyID,
		Code:      "tk",
		Name:      "Teknik Komputer",
		Level:     "S0",
	}
	_, err = svc.Create(ctx, reqBadLevel)
	if err == nil {
		t.Errorf("expected error on invalid level, got nil")
	}
}

func TestStudyProgramService_UpdateAndStatus(t *testing.T) {
	repo := newMockSPRepo()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := study_programs.NewService(repo, logger)
	ctx := context.Background()

	facID := uuid.New()
	repo.faculties[facID] = true

	created, err := svc.Create(ctx, study_programs.CreateStudyProgramRequest{
		FacultyID: facID,
		Code:      "SI",
		Name:      "Sistem Informasi",
		Level:     "S1",
	})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Update name
	newName := "Sistem Informasi Enterprise"
	updated, err := svc.Update(ctx, created.ID, study_programs.UpdateStudyProgramRequest{Name: &newName})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("expected updated name, got %s", updated.Name)
	}

	// Update status
	inactive := false
	statusRes, err := svc.UpdateStatus(ctx, created.ID, inactive)
	if err != nil {
		t.Fatalf("status update failed: %v", err)
	}
	if statusRes.IsActive != false {
		t.Errorf("expected status false, got %v", statusRes.IsActive)
	}
}
