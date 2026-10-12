package study_programs

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

var allowedSortFields = []string{"code", "name", "level", "created_at", "is_active"}

var validLevels = map[string]bool{
	"D3":      true,
	"D4":      true,
	"S1":      true,
	"S2":      true,
	"S3":      true,
	"Profesi": true,
}

// Service defines the business operations for study programs.
type Service struct {
	repo   Repository
	logger *slog.Logger
}

// NewService instantiates a new study program service.
func NewService(repo Repository, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// Create creates a new study program after validating business rules.
func (s *Service) Create(ctx context.Context, req CreateStudyProgramRequest) (*StudyProgramResponse, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	name := strings.TrimSpace(req.Name)
	level := strings.TrimSpace(req.Level)

	if code == "" {
		return nil, appErrors.ErrBadRequest("Kode program studi tidak boleh kosong")
	}
	if name == "" {
		return nil, appErrors.ErrBadRequest("Nama program studi tidak boleh kosong")
	}
	if !validLevels[level] {
		return nil, appErrors.ErrBadRequest("Jenjang pendidikan tidak valid. Pilihan: D3, D4, S1, S2, S3, Profesi")
	}

	// Verify parent faculty exists and is active
	exists, active, err := s.repo.CheckFacultyActive(ctx, req.FacultyID)
	if err != nil {
		return nil, appErrors.ErrInternal(err)
	}
	if !exists {
		return nil, appErrors.ErrNotFound(fmt.Sprintf("Fakultas dengan ID '%s' tidak ditemukan", req.FacultyID))
	}
	if !active {
		return nil, appErrors.ErrConflict("Fakultas induk sedang tidak aktif, tidak dapat menambahkan program studi baru")
	}

	// Check code uniqueness
	existing, err := s.repo.GetByCode(ctx, code)
	if err == nil && existing != nil {
		return nil, appErrors.ErrConflict(fmt.Sprintf("Program studi dengan kode '%s' sudah terdaftar", code))
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

	sp := &StudyProgram{
		FacultyID:   req.FacultyID,
		Code:        code,
		Name:        name,
		Level:       level,
		Description: desc,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, sp); err != nil {
		if errors.Is(err, ErrDuplicateCode) {
			return nil, appErrors.ErrConflict(fmt.Sprintf("Program studi dengan kode '%s' sudah terdaftar", code))
		}
		if errors.Is(err, ErrFacultyNotFound) {
			return nil, appErrors.ErrNotFound("Fakultas induk tidak ditemukan")
		}
		return nil, appErrors.ErrInternal(err)
	}

	// Load detail with faculty join
	created, err := s.repo.GetByID(ctx, sp.ID)
	if err != nil {
		return nil, appErrors.ErrInternal(err)
	}

	s.logger.Info("Study program created successfully",
		slog.String("study_program_id", created.ID.String()),
		slog.String("code", created.Code),
	)

	return mapToResponse(created), nil
}

// GetByID retrieves a study program by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*StudyProgramResponse, error) {
	sp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound(fmt.Sprintf("Program studi dengan ID '%s' tidak ditemukan", id))
		}
		return nil, appErrors.ErrInternal(err)
	}

	return mapToResponse(sp), nil
}

// List retrieves paginated study programs with filtering and sorting.
func (s *Service) List(ctx context.Context, filter StudyProgramFilter, p pagination.Params) ([]StudyProgramResponse, pagination.Meta, error) {
	if err := p.ValidateSort(allowedSortFields); err != nil {
		return nil, pagination.Meta{}, err
	}

	items, total, err := s.repo.List(ctx, filter, p)
	if err != nil {
		return nil, pagination.Meta{}, appErrors.ErrInternal(err)
	}

	responses := make([]StudyProgramResponse, len(items))
	for i := range items {
		responses[i] = *mapToResponse(&items[i])
	}

	meta := pagination.NewMeta(p.Page, p.Limit, total)
	return responses, meta, nil
}

// Update updates fields of an existing study program.
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateStudyProgramRequest) (*StudyProgramResponse, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound(fmt.Sprintf("Program studi dengan ID '%s' tidak ditemukan", id))
		}
		return nil, appErrors.ErrInternal(err)
	}

	updated := false

	if req.FacultyID != nil && *req.FacultyID != existing.FacultyID {
		exists, active, err := s.repo.CheckFacultyActive(ctx, *req.FacultyID)
		if err != nil {
			return nil, appErrors.ErrInternal(err)
		}
		if !exists {
			return nil, appErrors.ErrNotFound(fmt.Sprintf("Fakultas baru dengan ID '%s' tidak ditemukan", *req.FacultyID))
		}
		if !active {
			return nil, appErrors.ErrConflict("Fakultas tujuan sedang tidak aktif")
		}
		existing.FacultyID = *req.FacultyID
		updated = true
	}

	if req.Code != nil {
		newCode := strings.ToUpper(strings.TrimSpace(*req.Code))
		if newCode == "" {
			return nil, appErrors.ErrBadRequest("Kode program studi tidak boleh kosong")
		}
		if newCode != existing.Code {
			other, err := s.repo.GetByCode(ctx, newCode)
			if err == nil && other != nil && other.ID != id {
				return nil, appErrors.ErrConflict(fmt.Sprintf("Program studi dengan kode '%s' sudah terdaftar", newCode))
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
			return nil, appErrors.ErrBadRequest("Nama program studi tidak boleh kosong")
		}
		if newName != existing.Name {
			existing.Name = newName
			updated = true
		}
	}

	if req.Level != nil {
		newLevel := strings.TrimSpace(*req.Level)
		if !validLevels[newLevel] {
			return nil, appErrors.ErrBadRequest("Jenjang pendidikan tidak valid. Pilihan: D3, D4, S1, S2, S3, Profesi")
		}
		if newLevel != existing.Level {
			existing.Level = newLevel
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
				return nil, appErrors.ErrConflict("Kode program studi sudah terdaftar")
			}
			return nil, appErrors.ErrInternal(err)
		}
		s.logger.Info("Study program updated successfully", slog.String("study_program_id", id.String()))
	}

	refreshed, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, appErrors.ErrInternal(err)
	}

	return mapToResponse(refreshed), nil
}

// UpdateStatus changes the lifecycle active status of a study program.
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, isActive bool) (*StudyProgramResponse, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound(fmt.Sprintf("Program studi dengan ID '%s' tidak ditemukan", id))
		}
		return nil, appErrors.ErrInternal(err)
	}

	if existing.IsActive == isActive {
		return mapToResponse(existing), nil
	}

	if err := s.repo.UpdateStatus(ctx, id, isActive); err != nil {
		return nil, appErrors.ErrInternal(err)
	}

	s.logger.Info("Study program status changed",
		slog.String("study_program_id", id.String()),
		slog.Bool("is_active", isActive),
	)

	existing.IsActive = isActive
	existing.UpdatedAt = time.Now().UTC()

	return mapToResponse(existing), nil
}

func mapToResponse(sp *StudyProgram) *StudyProgramResponse {
	return &StudyProgramResponse{
		ID: sp.ID,
		Faculty: FacultySummary{
			ID:   sp.FacultyID,
			Code: sp.FacultyCode,
			Name: sp.FacultyName,
		},
		Code:        sp.Code,
		Name:        sp.Name,
		Level:       sp.Level,
		Description: sp.Description,
		IsActive:    sp.IsActive,
		CreatedAt:   sp.CreatedAt,
		UpdatedAt:   sp.UpdatedAt,
	}
}
