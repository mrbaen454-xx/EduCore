package faculties_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"educore/internal/modules/faculties"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
)

type mockRepo struct {
	faculties map[uuid.UUID]*faculties.Faculty
	byCode    map[string]*faculties.Faculty
	counts    map[uuid.UUID]int
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		faculties: make(map[uuid.UUID]*faculties.Faculty),
		byCode:    make(map[string]*faculties.Faculty),
		counts:    make(map[uuid.UUID]int),
	}
}

func (m *mockRepo) Create(ctx context.Context, f *faculties.Faculty) error {
	if _, exists := m.byCode[f.Code]; exists {
		return faculties.ErrDuplicateCode
	}
	f.ID = uuid.New()
	m.faculties[f.ID] = f
	m.byCode[f.Code] = f
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (*faculties.Faculty, error) {
	f, exists := m.faculties[id]
	if !exists {
		return nil, faculties.ErrNotFound
	}
	return f, nil
}

func (m *mockRepo) GetByCode(ctx context.Context, code string) (*faculties.Faculty, error) {
	f, exists := m.byCode[code]
	if !exists {
		return nil, faculties.ErrNotFound
	}
	return f, nil
}

func (m *mockRepo) List(ctx context.Context, filter faculties.FacultyFilter, p pagination.Params) ([]faculties.Faculty, int, error) {
	res := make([]faculties.Faculty, 0, len(m.faculties))
	for _, f := range m.faculties {
		if filter.IsActive != nil && f.IsActive != *filter.IsActive {
			continue
		}
		res = append(res, *f)
	}
	return res, len(res), nil
}

func (m *mockRepo) Update(ctx context.Context, f *faculties.Faculty) error {
	if _, exists := m.faculties[f.ID]; !exists {
		return faculties.ErrNotFound
	}
	m.faculties[f.ID] = f
	m.byCode[f.Code] = f
	return nil
}

func (m *mockRepo) UpdateStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	f, exists := m.faculties[id]
	if !exists {
		return faculties.ErrNotFound
	}
	f.IsActive = isActive
	return nil
}

func (m *mockRepo) CountStudyPrograms(ctx context.Context, id uuid.UUID) (int, error) {
	return m.counts[id], nil
}

func TestFacultyService_Create(t *testing.T) {
	repo := newMockRepo()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := faculties.NewService(repo, logger)

	ctx := context.Background()

	// 1. Valid creation
	req := faculties.CreateFacultyRequest{
		Code: "fik",
		Name: "Fakultas Ilmu Komputer",
	}
	res, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("expected create success, got: %v", err)
	}
	if res.Code != "FIK" { // normalized uppercase
		t.Errorf("expected code FIK, got %s", res.Code)
	}
	if res.Name != "Fakultas Ilmu Komputer" {
		t.Errorf("expected name match, got %s", res.Name)
	}
	if !res.IsActive {
		t.Errorf("expected new faculty to be active")
	}

	// 2. Duplicate code
	_, err = svc.Create(ctx, req)
	if err == nil {
		t.Fatalf("expected error for duplicate code, got nil")
	}
	appErr, ok := err.(*appErrors.AppError)
	if !ok || appErr.Code != appErrors.CodeConflict {
		t.Errorf("expected CONFLICT code, got %v", err)
	}

	// 3. Empty fields
	_, err = svc.Create(ctx, faculties.CreateFacultyRequest{Code: "", Name: "Test"})
	if err == nil {
		t.Errorf("expected error on empty code, got nil")
	}
	_, err = svc.Create(ctx, faculties.CreateFacultyRequest{Code: "FT", Name: "   "})
	if err == nil {
		t.Errorf("expected error on empty name, got nil")
	}
}

func TestFacultyService_GetByID(t *testing.T) {
	repo := newMockRepo()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := faculties.NewService(repo, logger)
	ctx := context.Background()

	f := &faculties.Faculty{
		ID:        uuid.New(),
		Code:      "FEB",
		Name:      "Fakultas Ekonomi dan Bisnis",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = repo.Create(ctx, f)

	// Found
	res, err := svc.GetByID(ctx, f.ID)
	if err != nil {
		t.Fatalf("expected get success, got: %v", err)
	}
	if res.Code != "FEB" {
		t.Errorf("expected code FEB, got %s", res.Code)
	}

	// Not Found
	_, err = svc.GetByID(ctx, uuid.New())
	if err == nil {
		t.Fatalf("expected not found error, got nil")
	}
	appErr, ok := err.(*appErrors.AppError)
	if !ok || appErr.Code != appErrors.CodeResourceNotFound {
		t.Errorf("expected RESOURCE_NOT_FOUND code, got %v", err)
	}
}

func TestFacultyService_UpdateAndStatus(t *testing.T) {
	repo := newMockRepo()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := faculties.NewService(repo, logger)
	ctx := context.Background()

	res, err := svc.Create(ctx, faculties.CreateFacultyRequest{Code: "FK", Name: "Fakultas Kedokteran"})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Update name
	newName := "Fakultas Kedokteran dan Kesehatan"
	updated, err := svc.Update(ctx, res.ID, faculties.UpdateFacultyRequest{Name: &newName})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("expected updated name, got %s", updated.Name)
	}

	// Update status
	inactive := false
	statusRes, err := svc.UpdateStatus(ctx, res.ID, inactive)
	if err != nil {
		t.Fatalf("status update failed: %v", err)
	}
	if statusRes.IsActive != false {
		t.Errorf("expected status false, got %v", statusRes.IsActive)
	}
}
