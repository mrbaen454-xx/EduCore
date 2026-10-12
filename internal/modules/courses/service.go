package courses

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

var allowedSortColumns = []string{"code", "name", "credits", "created_at"}

// Service defines business operations for courses.
type Service interface {
	Create(ctx context.Context, req CreateCourseRequest) (*CourseResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*CourseResponse, error)
	List(ctx context.Context, filter ListCoursesFilter, p pagination.Params) ([]CourseResponse, pagination.Meta, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateCourseRequest) (*CourseResponse, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, req UpdateCourseStatusRequest) (*CourseResponse, error)
	AddPrerequisite(ctx context.Context, courseID uuid.UUID, req AddPrerequisiteRequest) error
	RemovePrerequisite(ctx context.Context, courseID, prereqID uuid.UUID) error
}

type service struct {
	repo   Repository
	logger *slog.Logger
}

// NewService creates a new Service instance for courses.
func NewService(repo Repository, logger *slog.Logger) Service {
	return &service{
		repo:   repo,
		logger: logger,
	}
}

func (s *service) Create(ctx context.Context, req CreateCourseRequest) (*CourseResponse, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		return nil, appErrors.ErrBadRequest("Kode mata kuliah tidak boleh kosong")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, appErrors.ErrBadRequest("Nama mata kuliah tidak boleh kosong")
	}

	if req.Credits < 1 || req.Credits > 8 {
		return nil, appErrors.ErrBadRequest("Jumlah SKS harus bernilai antara 1 dan 8")
	}

	var desc *string
	if req.Description != nil {
		d := strings.TrimSpace(*req.Description)
		desc = &d
	}

	now := time.Now().UTC()
	c := &Course{
		Code:        code,
		Name:        name,
		Description: desc,
		Credits:     req.Credits,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, c); err != nil {
		if errors.Is(err, ErrDuplicateCode) {
			return nil, appErrors.ErrConflict(fmt.Sprintf("Kode mata kuliah '%s' sudah digunakan", code))
		}
		s.logger.Error("Failed to create course", slog.Any("error", err), slog.String("code", code))
		return nil, appErrors.ErrInternal(err)
	}

	s.logger.Info("Course created successfully", slog.String("id", c.ID.String()), slog.String("code", c.Code))
	return s.toResponse(c, nil), nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*CourseResponse, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound("Mata kuliah tidak ditemukan")
		}
		s.logger.Error("Failed to get course", slog.Any("error", err), slog.String("id", id.String()))
		return nil, appErrors.ErrInternal(err)
	}

	prereqs, err := s.repo.GetPrerequisites(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get prerequisites for course", slog.Any("error", err), slog.String("id", id.String()))
		return nil, appErrors.ErrInternal(err)
	}

	return s.toResponse(c, prereqs), nil
}

func (s *service) List(ctx context.Context, filter ListCoursesFilter, p pagination.Params) ([]CourseResponse, pagination.Meta, error) {
	p.ValidateSort(allowedSortColumns)

	list, total, err := s.repo.List(ctx, filter, p)
	if err != nil {
		s.logger.Error("Failed to list courses", slog.Any("error", err))
		return nil, pagination.Meta{}, appErrors.ErrInternal(err)
	}

	res := make([]CourseResponse, 0, len(list))
	for i := range list {
		res = append(res, *s.toResponse(&list[i], nil))
	}

	meta := pagination.NewMeta(total, p.Page, p.Limit)
	return res, meta, nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req UpdateCourseRequest) (*CourseResponse, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound("Mata kuliah tidak ditemukan")
		}
		return nil, appErrors.ErrInternal(err)
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, appErrors.ErrBadRequest("Nama mata kuliah tidak boleh kosong")
		}
		c.Name = name
	}

	if req.Description != nil {
		desc := strings.TrimSpace(*req.Description)
		c.Description = &desc
	}

	if req.Credits != nil {
		if *req.Credits < 1 || *req.Credits > 8 {
			return nil, appErrors.ErrBadRequest("Jumlah SKS harus bernilai antara 1 dan 8")
		}
		c.Credits = *req.Credits
	}

	c.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, c); err != nil {
		s.logger.Error("Failed to update course", slog.Any("error", err), slog.String("id", id.String()))
		return nil, appErrors.ErrInternal(err)
	}

	prereqs, _ := s.repo.GetPrerequisites(ctx, id)
	return s.toResponse(c, prereqs), nil
}

func (s *service) UpdateStatus(ctx context.Context, id uuid.UUID, req UpdateCourseStatusRequest) (*CourseResponse, error) {
	if req.IsActive == nil {
		return nil, appErrors.ErrBadRequest("Status is_active wajib diisi")
	}

	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound("Mata kuliah tidak ditemukan")
		}
		return nil, appErrors.ErrInternal(err)
	}

	if err := s.repo.UpdateStatus(ctx, id, *req.IsActive); err != nil {
		s.logger.Error("Failed to update course status", slog.Any("error", err), slog.String("id", id.String()))
		return nil, appErrors.ErrInternal(err)
	}

	c.IsActive = *req.IsActive
	c.UpdatedAt = time.Now().UTC()

	prereqs, _ := s.repo.GetPrerequisites(ctx, id)
	return s.toResponse(c, prereqs), nil
}

func (s *service) AddPrerequisite(ctx context.Context, courseID uuid.UUID, req AddPrerequisiteRequest) error {
	if courseID == req.PrerequisiteCourseID {
		return appErrors.ErrBadRequest("Mata kuliah tidak dapat menjadi prasyarat bagi dirinya sendiri")
	}

	// Verify target course exists
	_, err := s.repo.GetByID(ctx, courseID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return appErrors.ErrNotFound("Mata kuliah target tidak ditemukan")
		}
		return appErrors.ErrInternal(err)
	}

	// Verify prerequisite course exists
	prereqCourse, err := s.repo.GetByID(ctx, req.PrerequisiteCourseID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return appErrors.ErrNotFound("Mata kuliah prasyarat tidak ditemukan")
		}
		return appErrors.ErrInternal(err)
	}

	if !prereqCourse.IsActive {
		return appErrors.ErrConflict("Mata kuliah prasyarat dalam kondisi tidak aktif")
	}

	// Circular dependency check: does prereqCourse already have courseID as its prerequisite?
	isCycle, err := s.repo.IsPrerequisite(ctx, req.PrerequisiteCourseID, courseID)
	if err != nil {
		return appErrors.ErrInternal(err)
	}
	if isCycle {
		return appErrors.ErrConflict("Terdeteksi siklus prasyarat antara kedua mata kuliah")
	}

	if err := s.repo.AddPrerequisite(ctx, courseID, req.PrerequisiteCourseID); err != nil {
		if errors.Is(err, ErrDuplicatePrereq) {
			return appErrors.ErrConflict("Prasyarat tersebut sudah terdaftar sebelumnya")
		}
		if errors.Is(err, ErrSelfPrereq) {
			return appErrors.ErrBadRequest("Mata kuliah tidak dapat menjadi prasyarat bagi dirinya sendiri")
		}
		s.logger.Error("Failed to add prerequisite", slog.Any("error", err))
		return appErrors.ErrInternal(err)
	}

	s.logger.Info("Prerequisite added",
		slog.String("course_id", courseID.String()),
		slog.String("prerequisite_id", req.PrerequisiteCourseID.String()))
	return nil
}

func (s *service) RemovePrerequisite(ctx context.Context, courseID, prereqID uuid.UUID) error {
	if err := s.repo.RemovePrerequisite(ctx, courseID, prereqID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return appErrors.ErrNotFound("Relasi prasyarat tidak ditemukan")
		}
		s.logger.Error("Failed to remove prerequisite", slog.Any("error", err))
		return appErrors.ErrInternal(err)
	}
	return nil
}

func (s *service) toResponse(c *Course, prereqs []PrerequisiteDetail) *CourseResponse {
	return &CourseResponse{
		ID:            c.ID,
		Code:          c.Code,
		Name:          c.Name,
		Description:   c.Description,
		Credits:       c.Credits,
		IsActive:      c.IsActive,
		Prerequisites: prereqs,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}
