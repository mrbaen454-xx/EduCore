package faculties

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
)

var allowedSortFields = []string{"code", "name", "created_at", "is_active"}

// Service defines the business logic operations for faculties.
type Service struct {
	repo   Repository
	logger *slog.Logger
}

// NewService instantiates a new faculty service.
func NewService(repo Repository, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// Create creates a new faculty after validating business rules.
func (s *Service) Create(ctx context.Context, req CreateFacultyRequest) (*FacultyResponse, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	name := strings.TrimSpace(req.Name)

	if code == "" {
		return nil, appErrors.ErrBadRequest("Kode fakultas tidak boleh kosong")
	}
	if name == "" {
		return nil, appErrors.ErrBadRequest("Nama fakultas tidak boleh kosong")
	}

	// Check existing code
	existing, err := s.repo.GetByCode(ctx, code)
	if err == nil && existing != nil {
		return nil, appErrors.ErrConflict(fmt.Sprintf("Fakultas dengan kode '%s' sudah terdaftar", code))
	} else if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, appErrors.ErrInternal(err)
	}

	now := time.Now().UTC()
	var desc *string
	if req.Description != nil {
		trimmed := strings.TrimSpace(*req.Description)
		if trimmed != "" {
			desc = &trimmed
		}
	}

	faculty := &Faculty{
		Code:        code,
		Name:        name,
		Description: desc,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, faculty); err != nil {
		if errors.Is(err, ErrDuplicateCode) {
			return nil, appErrors.ErrConflict(fmt.Sprintf("Fakultas dengan kode '%s' sudah terdaftar", code))
		}
		return nil, appErrors.ErrInternal(err)
	}

	s.logger.Info("Faculty created successfully",
		slog.String("faculty_id", faculty.ID.String()),
		slog.String("code", faculty.Code),
	)

	count := 0
	return &FacultyResponse{
		ID:                faculty.ID,
		Code:              faculty.Code,
		Name:              faculty.Name,
		Description:       faculty.Description,
		IsActive:          faculty.IsActive,
		StudyProgramCount: &count,
		CreatedAt:         faculty.CreatedAt,
		UpdatedAt:         faculty.UpdatedAt,
	}, nil
}

// GetByID retrieves a faculty by ID, including the count of linked study programs.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*FacultyResponse, error) {
	faculty, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound(fmt.Sprintf("Fakultas dengan ID '%s' tidak ditemukan", id))
		}
		return nil, appErrors.ErrInternal(err)
	}

	count, err := s.repo.CountStudyPrograms(ctx, id)
	if err != nil {
		s.logger.Warn("Failed to count linked study programs for faculty", slog.String("faculty_id", id.String()), slog.Any("error", err))
		count = 0
	}

	return &FacultyResponse{
		ID:                faculty.ID,
		Code:              faculty.Code,
		Name:              faculty.Name,
		Description:       faculty.Description,
		IsActive:          faculty.IsActive,
		StudyProgramCount: &count,
		CreatedAt:         faculty.CreatedAt,
		UpdatedAt:         faculty.UpdatedAt,
	}, nil
}

// List retrieves paginated faculties with filtering and sorting.
func (s *Service) List(ctx context.Context, filter FacultyFilter, p pagination.Params) ([]FacultyResponse, pagination.Meta, error) {
	if err := p.ValidateSort(allowedSortFields); err != nil {
		return nil, pagination.Meta{}, err
	}

	faculties, total, err := s.repo.List(ctx, filter, p)
	if err != nil {
		return nil, pagination.Meta{}, appErrors.ErrInternal(err)
	}

	responses := make([]FacultyResponse, len(faculties))
	for i, f := range faculties {
		responses[i] = FacultyResponse{
			ID:          f.ID,
			Code:        f.Code,
			Name:        f.Name,
			Description: f.Description,
			IsActive:    f.IsActive,
			CreatedAt:   f.CreatedAt,
			UpdatedAt:   f.UpdatedAt,
		}
	}

	meta := pagination.NewMeta(p.Page, p.Limit, total)
	return responses, meta, nil
}

// Update updates fields of an existing faculty.
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateFacultyRequest) (*FacultyResponse, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound(fmt.Sprintf("Fakultas dengan ID '%s' tidak ditemukan", id))
		}
		return nil, appErrors.ErrInternal(err)
	}

	updated := false

	if req.Code != nil {
		newCode := strings.ToUpper(strings.TrimSpace(*req.Code))
		if newCode == "" {
			return nil, appErrors.ErrBadRequest("Kode fakultas tidak boleh kosong")
		}
		if newCode != existing.Code {
			// Check if new code conflicts
			other, err := s.repo.GetByCode(ctx, newCode)
			if err == nil && other != nil && other.ID != id {
				return nil, appErrors.ErrConflict(fmt.Sprintf("Fakultas dengan kode '%s' sudah terdaftar", newCode))
			} else if err != nil && !errors.Is(err, ErrNotFound) {
				return nil, appErrors.ErrInternal(err)
			}
			existing.Code = newCode
			updated = true
		}
	}

	if req.Name != nil {
		newName := strings.TrimSpace(*req.Name)
		if newName == "" {
			return nil, appErrors.ErrBadRequest("Nama fakultas tidak boleh kosong")
		}
		if newName != existing.Name {
			existing.Name = newName
			updated = true
		}
	}

	if req.Description != nil {
		newDesc := strings.TrimSpace(*req.Description)
		if newDesc == "" {
			existing.Description = nil
		} else {
			existing.Description = &newDesc
		}
		updated = true
	}

	if updated {
		existing.UpdatedAt = time.Now().UTC()
		if err := s.repo.Update(ctx, existing); err != nil {
			if errors.Is(err, ErrDuplicateCode) {
				return nil, appErrors.ErrConflict("Kode fakultas sudah terdaftar")
			}
			return nil, appErrors.ErrInternal(err)
		}
		s.logger.Info("Faculty updated successfully", slog.String("faculty_id", id.String()))
	}

	count, _ := s.repo.CountStudyPrograms(ctx, id)
	return &FacultyResponse{
		ID:                existing.ID,
		Code:              existing.Code,
		Name:              existing.Name,
		Description:       existing.Description,
		IsActive:          existing.IsActive,
		StudyProgramCount: &count,
		CreatedAt:         existing.CreatedAt,
		UpdatedAt:         existing.UpdatedAt,
	}, nil
}

// UpdateStatus changes the active lifecycle status of a faculty.
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, isActive bool) (*FacultyResponse, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound(fmt.Sprintf("Fakultas dengan ID '%s' tidak ditemukan", id))
		}
		return nil, appErrors.ErrInternal(err)
	}

	if existing.IsActive == isActive {
		count, _ := s.repo.CountStudyPrograms(ctx, id)
		return &FacultyResponse{
			ID:                existing.ID,
			Code:              existing.Code,
			Name:              existing.Name,
			Description:       existing.Description,
			IsActive:          existing.IsActive,
			StudyProgramCount: &count,
			CreatedAt:         existing.CreatedAt,
			UpdatedAt:         existing.UpdatedAt,
		}, nil
	}

	if err := s.repo.UpdateStatus(ctx, id, isActive); err != nil {
		return nil, appErrors.ErrInternal(err)
	}

	s.logger.Info("Faculty status changed",
		slog.String("faculty_id", id.String()),
		slog.Bool("is_active", isActive),
	)

	existing.IsActive = isActive
	existing.UpdatedAt = time.Now().UTC()
	count, _ := s.repo.CountStudyPrograms(ctx, id)

	return &FacultyResponse{
		ID:                existing.ID,
		Code:              existing.Code,
		Name:              existing.Name,
		Description:       existing.Description,
		IsActive:          existing.IsActive,
		StudyProgramCount: &count,
		CreatedAt:         existing.CreatedAt,
		UpdatedAt:         existing.UpdatedAt,
	}, nil
}
