package lecturers

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

var allowedSortFields = []string{"nidn", "name", "created_at", "status"}

var validStatuses = map[string]bool{
	"ACTIVE":   true,
	"INACTIVE": true,
	"LEAVE":    true,
}

// Service defines the business operations for lecturers.
type Service struct {
	repo   Repository
	logger *slog.Logger
}

// NewService instantiates a new lecturer service.
func NewService(repo Repository, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// Create creates a new lecturer after enforcing business integrity.
func (s *Service) Create(ctx context.Context, req CreateLecturerRequest) (*LecturerResponse, error) {
	nidn := strings.TrimSpace(req.NIDN)
	name := strings.TrimSpace(req.Name)
	email := strings.ToLower(strings.TrimSpace(req.Email))

	if nidn == "" {
		return nil, appErrors.ErrBadRequest("NIDN dosen tidak boleh kosong")
	}
	if name == "" {
		return nil, appErrors.ErrBadRequest("Nama dosen tidak boleh kosong")
	}
	if email == "" {
		return nil, appErrors.ErrBadRequest("Email dosen tidak boleh kosong")
	}

	// Verify study program exists and is active
	exists, active, err := s.repo.CheckStudyProgramActive(ctx, req.StudyProgramID)
	if err != nil {
		return nil, appErrors.ErrInternal(err)
	}
	if !exists {
		return nil, appErrors.ErrNotFound(fmt.Sprintf("Program studi dengan ID '%s' tidak ditemukan", req.StudyProgramID))
	}
	if !active {
		return nil, appErrors.ErrConflict("Program studi sedang tidak aktif, tidak dapat mendaftarkan dosen baru")
	}

	// Verify user if provided
	if req.UserID != nil {
		userExists, err := s.repo.CheckUserExists(ctx, *req.UserID)
		if err != nil {
			return nil, appErrors.ErrInternal(err)
		}
		if !userExists {
			return nil, appErrors.ErrNotFound(fmt.Sprintf("Akun pengguna dengan ID '%s' tidak ditemukan", *req.UserID))
		}
	}

	// Verify NIDN uniqueness
	existingNIDN, err := s.repo.GetByNIDN(ctx, nidn)
	if err == nil && existingNIDN != nil {
		return nil, appErrors.ErrConflict(fmt.Sprintf("Dosen dengan NIDN '%s' sudah terdaftar", nidn))
	} else if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, appErrors.ErrInternal(err)
	}

	// Verify Email uniqueness
	existingEmail, err := s.repo.GetByEmail(ctx, email)
	if err == nil && existingEmail != nil {
		return nil, appErrors.ErrConflict(fmt.Sprintf("Dosen dengan email '%s' sudah terdaftar", email))
	} else if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, appErrors.ErrInternal(err)
	}

	now := time.Now().UTC()
	var nip *string
	if req.NIP != nil {
		t := strings.TrimSpace(*req.NIP)
		if t != "" {
			nip = &t
		}
	}
	var prefix *string
	if req.TitlePrefix != nil {
		t := strings.TrimSpace(*req.TitlePrefix)
		if t != "" {
			prefix = &t
		}
	}
	var suffix *string
	if req.TitleSuffix != nil {
		t := strings.TrimSpace(*req.TitleSuffix)
		if t != "" {
			suffix = &t
		}
	}
	var phone *string
	if req.Phone != nil {
		t := strings.TrimSpace(*req.Phone)
		if t != "" {
			phone = &t
		}
	}

	lecturer := &Lecturer{
		UserID:         req.UserID,
		StudyProgramID: req.StudyProgramID,
		NIDN:           nidn,
		NIP:            nip,
		Name:           name,
		TitlePrefix:    prefix,
		TitleSuffix:    suffix,
		Email:          email,
		Phone:          phone,
		Status:         "ACTIVE",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, lecturer); err != nil {
		if errors.Is(err, ErrDuplicateNIDN) {
			return nil, appErrors.ErrConflict(fmt.Sprintf("Dosen dengan NIDN '%s' sudah terdaftar", nidn))
		}
		if errors.Is(err, ErrDuplicateEmail) {
			return nil, appErrors.ErrConflict(fmt.Sprintf("Dosen dengan email '%s' sudah terdaftar", email))
		}
		if errors.Is(err, ErrUserAlreadyLinked) {
			return nil, appErrors.ErrConflict("Akun pengguna sudah terhubung dengan dosen lain")
		}
		return nil, appErrors.ErrInternal(err)
	}

	created, err := s.repo.GetByID(ctx, lecturer.ID)
	if err != nil {
		return nil, appErrors.ErrInternal(err)
	}

	s.logger.Info("Lecturer registered successfully",
		slog.String("lecturer_id", created.ID.String()),
		slog.String("nidn", created.NIDN),
	)

	return mapToResponse(created), nil
}

// GetByID retrieves a lecturer by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*LecturerResponse, error) {
	l, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound(fmt.Sprintf("Dosen dengan ID '%s' tidak ditemukan", id))
		}
		return nil, appErrors.ErrInternal(err)
	}

	return mapToResponse(l), nil
}

// List retrieves paginated lecturers with filtering and sorting.
func (s *Service) List(ctx context.Context, filter LecturerFilter, p pagination.Params) ([]LecturerResponse, pagination.Meta, error) {
	if err := p.ValidateSort(allowedSortFields); err != nil {
		return nil, pagination.Meta{}, err
	}

	items, total, err := s.repo.List(ctx, filter, p)
	if err != nil {
		return nil, pagination.Meta{}, appErrors.ErrInternal(err)
	}

	responses := make([]LecturerResponse, len(items))
	for i := range items {
		responses[i] = *mapToResponse(&items[i])
	}

	meta := pagination.NewMeta(p.Page, p.Limit, total)
	return responses, meta, nil
}

// Update updates fields of an existing lecturer.
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateLecturerRequest) (*LecturerResponse, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound(fmt.Sprintf("Dosen dengan ID '%s' tidak ditemukan", id))
		}
		return nil, appErrors.ErrInternal(err)
	}

	updated := false

	if req.StudyProgramID != nil && *req.StudyProgramID != existing.StudyProgramID {
		exists, active, err := s.repo.CheckStudyProgramActive(ctx, *req.StudyProgramID)
		if err != nil {
			return nil, appErrors.ErrInternal(err)
		}
		if !exists {
			return nil, appErrors.ErrNotFound("Program studi baru tidak ditemukan")
		}
		if !active {
			return nil, appErrors.ErrConflict("Program studi tujuan sedang tidak aktif")
		}
		existing.StudyProgramID = *req.StudyProgramID
		updated = true
	}

	if req.UserID != nil {
		if existing.UserID == nil || *req.UserID != *existing.UserID {
			userExists, err := s.repo.CheckUserExists(ctx, *req.UserID)
			if err != nil {
				return nil, appErrors.ErrInternal(err)
			}
			if !userExists {
				return nil, appErrors.ErrNotFound("Akun pengguna tidak ditemukan")
			}
			existing.UserID = req.UserID
			updated = true
		}
	}

	if req.NIDN != nil {
		newNIDN := strings.TrimSpace(*req.NIDN)
		if newNIDN == "" {
			return nil, appErrors.ErrBadRequest("NIDN tidak boleh kosong")
		}
		if newNIDN != existing.NIDN {
			other, err := s.repo.GetByNIDN(ctx, newNIDN)
			if err == nil && other != nil && other.ID != id {
				return nil, appErrors.ErrConflict(fmt.Sprintf("Dosen dengan NIDN '%s' sudah terdaftar", newNIDN))
			} else if err != nil && !errors.Is(err, ErrNotFound) {
				return nil, appErrors.ErrInternal(err)
			}
			existing.NIDN = newNIDN
			updated = true
		}
	}

	if req.Email != nil {
		newEmail := strings.ToLower(strings.TrimSpace(*req.Email))
		if newEmail == "" {
			return nil, appErrors.ErrBadRequest("Email tidak boleh kosong")
		}
		if newEmail != existing.Email {
			other, err := s.repo.GetByEmail(ctx, newEmail)
			if err == nil && other != nil && other.ID != id {
				return nil, appErrors.ErrConflict(fmt.Sprintf("Dosen dengan email '%s' sudah terdaftar", newEmail))
			} else if err != nil && !errors.Is(err, ErrNotFound) {
				return nil, appErrors.ErrInternal(err)
			}
			existing.Email = newEmail
			updated = true
		}
	}

	if req.Name != nil {
		newName := strings.TrimSpace(*req.Name)
		if newName == "" {
			return nil, appErrors.ErrBadRequest("Nama dosen tidak boleh kosong")
		}
		if newName != existing.Name {
			existing.Name = newName
			updated = true
		}
	}

	if req.NIP != nil {
		t := strings.TrimSpace(*req.NIP)
		if t == "" {
			existing.NIP = nil
		} else {
			existing.NIP = &t
		}
		updated = true
	}

	if req.TitlePrefix != nil {
		t := strings.TrimSpace(*req.TitlePrefix)
		if t == "" {
			existing.TitlePrefix = nil
		} else {
			existing.TitlePrefix = &t
		}
		updated = true
	}

	if req.TitleSuffix != nil {
		t := strings.TrimSpace(*req.TitleSuffix)
		if t == "" {
			existing.TitleSuffix = nil
		} else {
			existing.TitleSuffix = &t
		}
		updated = true
	}

	if req.Phone != nil {
		t := strings.TrimSpace(*req.Phone)
		if t == "" {
			existing.Phone = nil
		} else {
			existing.Phone = &t
		}
		updated = true
	}

	if updated {
		existing.UpdatedAt = time.Now().UTC()
		if err := s.repo.Update(ctx, existing); err != nil {
			if errors.Is(err, ErrDuplicateNIDN) {
				return nil, appErrors.ErrConflict("NIDN sudah terdaftar")
			}
			if errors.Is(err, ErrDuplicateEmail) {
				return nil, appErrors.ErrConflict("Email sudah terdaftar")
			}
			if errors.Is(err, ErrUserAlreadyLinked) {
				return nil, appErrors.ErrConflict("Akun pengguna sudah terhubung dengan dosen lain")
			}
			return nil, appErrors.ErrInternal(err)
		}
		s.logger.Info("Lecturer updated successfully", slog.String("lecturer_id", id.String()))
	}

	refreshed, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, appErrors.ErrInternal(err)
	}

	return mapToResponse(refreshed), nil
}

// UpdateStatus changes the active lifecycle status of a lecturer.
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*LecturerResponse, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	if !validStatuses[status] {
		return nil, appErrors.ErrBadRequest("Status dosen tidak valid. Pilihan: ACTIVE, INACTIVE, LEAVE")
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound(fmt.Sprintf("Dosen dengan ID '%s' tidak ditemukan", id))
		}
		return nil, appErrors.ErrInternal(err)
	}

	if existing.Status == status {
		return mapToResponse(existing), nil
	}

	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return nil, appErrors.ErrInternal(err)
	}

	s.logger.Info("Lecturer status changed",
		slog.String("lecturer_id", id.String()),
		slog.String("status", status),
	)

	existing.Status = status
	existing.UpdatedAt = time.Now().UTC()

	return mapToResponse(existing), nil
}

func mapToResponse(l *Lecturer) *LecturerResponse {
	return &LecturerResponse{
		ID:     l.ID,
		UserID: l.UserID,
		StudyProgram: StudyProgramSummary{
			ID:   l.StudyProgramID,
			Code: l.StudyProgramCode,
			Name: l.StudyProgramName,
		},
		NIDN:        l.NIDN,
		NIP:         l.NIP,
		Name:        l.Name,
		FullName:    l.FullNameWithTitles(),
		TitlePrefix: l.TitlePrefix,
		TitleSuffix: l.TitleSuffix,
		Email:       l.Email,
		Phone:       l.Phone,
		Status:      l.Status,
		CreatedAt:   l.CreatedAt,
		UpdatedAt:   l.UpdatedAt,
	}
}
