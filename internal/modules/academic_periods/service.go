package academic_periods

import (
	"context"
	"log/slog"
	"strings"
	"time"

	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
)

type Service interface {
	Create(ctx context.Context, req CreateAcademicPeriodRequest) (*AcademicPeriodResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*AcademicPeriodResponse, error)
	List(ctx context.Context, filter AcademicPeriodFilter, p pagination.Params) ([]AcademicPeriodResponse, int, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateAcademicPeriodRequest) (*AcademicPeriodResponse, error)
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

func (s *service) Create(ctx context.Context, req CreateAcademicPeriodRequest) (*AcademicPeriodResponse, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	name := strings.TrimSpace(req.Name)

	if code == "" || name == "" {
		return nil, appErrors.ErrBadRequest("Kode dan nama periode akademik wajib diisi")
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, appErrors.ErrBadRequest("Format start_date tidak valid (gunakan YYYY-MM-DD)")
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, appErrors.ErrBadRequest("Format end_date tidak valid (gunakan YYYY-MM-DD)")
	}

	if startDate.After(endDate) {
		return nil, appErrors.ErrBadRequest("Tanggal mulai tidak boleh melebihi tanggal selesai")
	}

	isActive := false
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	p := &AcademicPeriod{
		Code:         code,
		Name:         name,
		SemesterType: req.SemesterType,
		StartDate:    startDate,
		EndDate:      endDate,
		IsActive:     isActive,
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}

	resp := ToResponse(p)
	return &resp, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*AcademicPeriodResponse, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := ToResponse(p)
	return &resp, nil
}

func (s *service) List(ctx context.Context, filter AcademicPeriodFilter, p pagination.Params) ([]AcademicPeriodResponse, int, error) {
	list, total, err := s.repo.List(ctx, filter, p)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]AcademicPeriodResponse, len(list))
	for i, item := range list {
		responses[i] = ToResponse(&item)
	}

	return responses, total, nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req UpdateAcademicPeriodRequest) (*AcademicPeriodResponse, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, appErrors.ErrBadRequest("Nama periode akademik tidak boleh kosong")
		}
		p.Name = name
	}

	if req.SemesterType != nil {
		p.SemesterType = *req.SemesterType
	}

	if req.StartDate != nil {
		sd, err := time.Parse("2006-01-02", *req.StartDate)
		if err != nil {
			return nil, appErrors.ErrBadRequest("Format start_date tidak valid (gunakan YYYY-MM-DD)")
		}
		p.StartDate = sd
	}

	if req.EndDate != nil {
		ed, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			return nil, appErrors.ErrBadRequest("Format end_date tidak valid (gunakan YYYY-MM-DD)")
		}
		p.EndDate = ed
	}

	if p.StartDate.After(p.EndDate) {
		return nil, appErrors.ErrBadRequest("Tanggal mulai tidak boleh melebihi tanggal selesai")
	}

	if req.IsActive != nil {
		p.IsActive = *req.IsActive
	}

	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}

	resp := ToResponse(p)
	return &resp, nil
}
