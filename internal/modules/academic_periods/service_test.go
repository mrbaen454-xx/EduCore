package academic_periods_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"educore/internal/modules/academic_periods"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
)

type mockAcademicPeriodRepo struct {
	periods map[uuid.UUID]*academic_periods.AcademicPeriod
	byCode  map[string]*academic_periods.AcademicPeriod
}

func newMockAcademicPeriodRepo() *mockAcademicPeriodRepo {
	return &mockAcademicPeriodRepo{
		periods: make(map[uuid.UUID]*academic_periods.AcademicPeriod),
		byCode:  make(map[string]*academic_periods.AcademicPeriod),
	}
}

func (m *mockAcademicPeriodRepo) Create(ctx context.Context, p *academic_periods.AcademicPeriod) error {
	if _, exists := m.byCode[p.Code]; exists {
		return academic_periods.ErrDuplicateCode
	}
	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	m.periods[p.ID] = p
	m.byCode[p.Code] = p
	return nil
}

func (m *mockAcademicPeriodRepo) GetByID(ctx context.Context, id uuid.UUID) (*academic_periods.AcademicPeriod, error) {
	p, exists := m.periods[id]
	if !exists {
		return nil, academic_periods.ErrNotFound
	}
	return p, nil
}

func (m *mockAcademicPeriodRepo) GetByCode(ctx context.Context, code string) (*academic_periods.AcademicPeriod, error) {
	p, exists := m.byCode[code]
	if !exists {
		return nil, academic_periods.ErrNotFound
	}
	return p, nil
}

func (m *mockAcademicPeriodRepo) List(ctx context.Context, filter academic_periods.AcademicPeriodFilter, p pagination.Params) ([]academic_periods.AcademicPeriod, int, error) {
	var list []academic_periods.AcademicPeriod
	for _, item := range m.periods {
		list = append(list, *item)
	}
	return list, len(list), nil
}

func (m *mockAcademicPeriodRepo) Update(ctx context.Context, p *academic_periods.AcademicPeriod) error {
	existing, exists := m.periods[p.ID]
	if !exists {
		return academic_periods.ErrNotFound
	}
	existing.Name = p.Name
	existing.SemesterType = p.SemesterType
	existing.StartDate = p.StartDate
	existing.EndDate = p.EndDate
	existing.IsActive = p.IsActive
	existing.UpdatedAt = time.Now()
	return nil
}

func TestAcademicPeriodService_Create(t *testing.T) {
	repo := newMockAcademicPeriodRepo()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := academic_periods.NewService(repo, logger)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		res, err := svc.Create(ctx, academic_periods.CreateAcademicPeriodRequest{
			Code:         "20241",
			Name:         "Semester Ganjil 2024/2025",
			SemesterType: "ODD",
			StartDate:    "2024-09-01",
			EndDate:      "2025-01-31",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Code != "20241" {
			t.Errorf("expected code 20241, got %s", res.Code)
		}
	})

	t.Run("Duplicate Code", func(t *testing.T) {
		_, err := svc.Create(ctx, academic_periods.CreateAcademicPeriodRequest{
			Code:         "20241",
			Name:         "Semester Ganjil Duplikat",
			SemesterType: "ODD",
			StartDate:    "2024-09-01",
			EndDate:      "2025-01-31",
		})
		if err == nil {
			t.Fatalf("expected conflict error, got nil")
		}
		appErr, ok := err.(*appErrors.AppError)
		if !ok || appErr.Code != appErrors.CodeConflict {
			t.Errorf("expected CONFLICT code, got %v", err)
		}
	})

	t.Run("Invalid Dates (Start After End)", func(t *testing.T) {
		_, err := svc.Create(ctx, academic_periods.CreateAcademicPeriodRequest{
			Code:         "20242",
			Name:         "Semester Genap Terbalik",
			SemesterType: "EVEN",
			StartDate:    "2025-06-01",
			EndDate:      "2025-02-01",
		})
		if err == nil {
			t.Fatalf("expected error for invalid dates")
		}
	})
}
