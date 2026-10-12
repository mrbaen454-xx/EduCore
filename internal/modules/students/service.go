package students

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

var allowedSortFields = []string{"nim", "name", "admission_year", "status", "created_at"}

var validStatuses = map[string]bool{
	"ACTIVE":    true,
	"INACTIVE":  true,
	"PROBATION": true,
	"GRADUATED": true,
	"DROPOUT":   true,
	"SUSPENDED": true,
}

var validGenders = map[string]bool{
	"MALE":   true,
	"FEMALE": true,
}

// Service defines the business operations for students.
type Service struct {
	repo   Repository
	logger *slog.Logger
}

// NewService instantiates a new student service.
func NewService(repo Repository, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// Create creates a new student profile after validating business rules.
func (s *Service) Create(ctx context.Context, req CreateStudentRequest) (*StudentResponse, error) {
	nim := strings.TrimSpace(req.NIM)
	name := strings.TrimSpace(req.Name)
	email := strings.ToLower(strings.TrimSpace(req.Email))

	if nim == "" {
		return nil, appErrors.ErrBadRequest("NIM mahasiswa tidak boleh kosong")
	}
	if name == "" {
		return nil, appErrors.ErrBadRequest("Nama mahasiswa tidak boleh kosong")
	}
	if email == "" {
		return nil, appErrors.ErrBadRequest("Email mahasiswa tidak boleh kosong")
	}
	if req.AdmissionYear < 1970 || req.AdmissionYear > 2100 {
		return nil, appErrors.ErrBadRequest("Tahun masuk tidak valid (antara 1970 dan 2100)")
	}

	var gender *string
	if req.Gender != nil {
		g := strings.ToUpper(strings.TrimSpace(*req.Gender))
		if !validGenders[g] {
			return nil, appErrors.ErrBadRequest("Jenis kelamin harus MALE atau FEMALE")
		}
		gender = &g
	}

	var birthDate *time.Time
	if req.BirthDate != nil {
		bdStr := strings.TrimSpace(*req.BirthDate)
		if bdStr != "" {
			parsed, err := time.Parse("2006-01-02", bdStr)
			if err != nil {
				return nil, appErrors.ErrBadRequest("Format tanggal lahir tidak valid (gunakan format YYYY-MM-DD)")
			}
			birthDate = &parsed
		}
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
		return nil, appErrors.ErrConflict("Program studi sedang tidak aktif, tidak dapat mendaftarkan mahasiswa baru")
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

	// Verify NIM uniqueness
	existingNIM, err := s.repo.GetByNIM(ctx, nim)
	if err == nil && existingNIM != nil {
		return nil, appErrors.ErrConflict(fmt.Sprintf("Mahasiswa dengan NIM '%s' sudah terdaftar", nim))
	} else if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, appErrors.ErrInternal(err)
	}

	// Verify Email uniqueness
	existingEmail, err := s.repo.GetByEmail(ctx, email)
	if err == nil && existingEmail != nil {
		return nil, appErrors.ErrConflict(fmt.Sprintf("Mahasiswa dengan email '%s' sudah terdaftar", email))
	} else if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, appErrors.ErrInternal(err)
	}

	now := time.Now().UTC()
	var phone *string
	if req.Phone != nil {
		t := strings.TrimSpace(*req.Phone)
		if t != "" {
			phone = &t
		}
	}

	student := &Student{
		UserID:         req.UserID,
		StudyProgramID: req.StudyProgramID,
		NIM:            nim,
		Name:           name,
		Email:          email,
		Phone:          phone,
		Gender:         gender,
		BirthDate:      birthDate,
		AdmissionYear:  req.AdmissionYear,
		Status:         "ACTIVE",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, student); err != nil {
		if errors.Is(err, ErrDuplicateNIM) {
			return nil, appErrors.ErrConflict(fmt.Sprintf("Mahasiswa dengan NIM '%s' sudah terdaftar", nim))
		}
		if errors.Is(err, ErrDuplicateEmail) {
			return nil, appErrors.ErrConflict(fmt.Sprintf("Mahasiswa dengan email '%s' sudah terdaftar", email))
		}
		if errors.Is(err, ErrUserAlreadyLinked) {
			return nil, appErrors.ErrConflict("Akun pengguna sudah terhubung dengan mahasiswa lain")
		}
		return nil, appErrors.ErrInternal(err)
	}

	created, err := s.repo.GetByID(ctx, student.ID)
	if err != nil {
		return nil, appErrors.ErrInternal(err)
	}

	s.logger.Info("Student registered successfully",
		slog.String("student_id", created.ID.String()),
		slog.String("nim", created.NIM),
	)

	return mapToResponse(created), nil
}

// GetByID retrieves a student by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*StudentResponse, error) {
	st, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound(fmt.Sprintf("Mahasiswa dengan ID '%s' tidak ditemukan", id))
		}
		return nil, appErrors.ErrInternal(err)
	}

	return mapToResponse(st), nil
}

// List retrieves paginated students with filtering and sorting.
func (s *Service) List(ctx context.Context, filter StudentFilter, p pagination.Params) ([]StudentResponse, pagination.Meta, error) {
	if err := p.ValidateSort(allowedSortFields); err != nil {
		return nil, pagination.Meta{}, err
	}

	items, total, err := s.repo.List(ctx, filter, p)
	if err != nil {
		return nil, pagination.Meta{}, appErrors.ErrInternal(err)
	}

	responses := make([]StudentResponse, len(items))
	for i := range items {
		responses[i] = *mapToResponse(&items[i])
	}

	meta := pagination.NewMeta(p.Page, p.Limit, total)
	return responses, meta, nil
}

// Update updates fields of an existing student.
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateStudentRequest) (*StudentResponse, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound(fmt.Sprintf("Mahasiswa dengan ID '%s' tidak ditemukan", id))
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
			return nil, appErrors.ErrNotFound("Program studi tujuan tidak ditemukan")
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

	if req.NIM != nil {
		newNIM := strings.TrimSpace(*req.NIM)
		if newNIM == "" {
			return nil, appErrors.ErrBadRequest("NIM tidak boleh kosong")
		}
		if newNIM != existing.NIM {
			other, err := s.repo.GetByNIM(ctx, newNIM)
			if err == nil && other != nil && other.ID != id {
				return nil, appErrors.ErrConflict(fmt.Sprintf("Mahasiswa dengan NIM '%s' sudah terdaftar", newNIM))
			} else if err != nil && !errors.Is(err, ErrNotFound) {
				return nil, appErrors.ErrInternal(err)
			}
			existing.NIM = newNIM
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
				return nil, appErrors.ErrConflict(fmt.Sprintf("Mahasiswa dengan email '%s' sudah terdaftar", newEmail))
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
			return nil, appErrors.ErrBadRequest("Nama mahasiswa tidak boleh kosong")
		}
		if newName != existing.Name {
			existing.Name = newName
			updated = true
		}
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

	if req.Gender != nil {
		g := strings.ToUpper(strings.TrimSpace(*req.Gender))
		if !validGenders[g] {
			return nil, appErrors.ErrBadRequest("Jenis kelamin harus MALE atau FEMALE")
		}
		existing.Gender = &g
		updated = true
	}

	if req.BirthDate != nil {
		bdStr := strings.TrimSpace(*req.BirthDate)
		if bdStr == "" {
			existing.BirthDate = nil
		} else {
			parsed, err := time.Parse("2006-01-02", bdStr)
			if err != nil {
				return nil, appErrors.ErrBadRequest("Format tanggal lahir tidak valid (gunakan format YYYY-MM-DD)")
			}
			existing.BirthDate = &parsed
		}
		updated = true
	}

	if req.AdmissionYear != nil {
		if *req.AdmissionYear < 1970 || *req.AdmissionYear > 2100 {
			return nil, appErrors.ErrBadRequest("Tahun masuk tidak valid (antara 1970 dan 2100)")
		}
		existing.AdmissionYear = *req.AdmissionYear
		updated = true
	}

	if updated {
		existing.UpdatedAt = time.Now().UTC()
		if err := s.repo.Update(ctx, existing); err != nil {
			if errors.Is(err, ErrDuplicateNIM) {
				return nil, appErrors.ErrConflict("NIM sudah terdaftar")
			}
			if errors.Is(err, ErrDuplicateEmail) {
				return nil, appErrors.ErrConflict("Email sudah terdaftar")
			}
			if errors.Is(err, ErrUserAlreadyLinked) {
				return nil, appErrors.ErrConflict("Akun pengguna sudah terhubung dengan mahasiswa lain")
			}
			return nil, appErrors.ErrInternal(err)
		}
		s.logger.Info("Student updated successfully", slog.String("student_id", id.String()))
	}

	refreshed, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, appErrors.ErrInternal(err)
	}

	return mapToResponse(refreshed), nil
}

// UpdateStatus changes the lifecycle active status of a student.
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*StudentResponse, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	if !validStatuses[status] {
		return nil, appErrors.ErrBadRequest("Status mahasiswa tidak valid. Pilihan: ACTIVE, INACTIVE, PROBATION, GRADUATED, DROPOUT, SUSPENDED")
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound(fmt.Sprintf("Mahasiswa dengan ID '%s' tidak ditemukan", id))
		}
		return nil, appErrors.ErrInternal(err)
	}

	if existing.Status == status {
		return mapToResponse(existing), nil
	}

	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return nil, appErrors.ErrInternal(err)
	}

	s.logger.Info("Student status changed",
		slog.String("student_id", id.String()),
		slog.String("status", status),
	)

	existing.Status = status
	existing.UpdatedAt = time.Now().UTC()

	return mapToResponse(existing), nil
}

func mapToResponse(s *Student) *StudentResponse {
	var birthDateStr *string
	if s.BirthDate != nil {
		formatted := s.BirthDate.Format("2006-01-02")
		birthDateStr = &formatted
	}

	return &StudentResponse{
		ID:     s.ID,
		UserID: s.UserID,
		StudyProgram: StudyProgramSummary{
			ID:   s.StudyProgramID,
			Code: s.StudyProgramCode,
			Name: s.StudyProgramName,
		},
		NIM:           s.NIM,
		Name:          s.Name,
		Email:         s.Email,
		Phone:         s.Phone,
		Gender:        s.Gender,
		BirthDate:     birthDateStr,
		AdmissionYear: s.AdmissionYear,
		Status:        s.Status,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}
