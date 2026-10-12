package curricula

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

var allowedCurriculaSortColumns = []string{"code", "name", "year", "created_at"}
var allowedCurriculumCoursesSortColumns = []string{"semester", "code", "name", "credits"}

// Service defines business operations for curricula and curriculum courses.
type Service interface {
	Create(ctx context.Context, req CreateCurriculumRequest) (*CurriculumResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*CurriculumResponse, error)
	List(ctx context.Context, filter ListCurriculaFilter, p pagination.Params) ([]CurriculumResponse, pagination.Meta, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateCurriculumRequest) (*CurriculumResponse, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, req UpdateCurriculumStatusRequest) (*CurriculumResponse, error)

	AddCourse(ctx context.Context, curriculumID uuid.UUID, req AddCurriculumCourseRequest) (*CurriculumCourseDetail, error)
	GetCourse(ctx context.Context, curriculumID, courseID uuid.UUID) (*CurriculumCourseDetail, error)
	ListCourses(ctx context.Context, curriculumID uuid.UUID, filter ListCurriculumCoursesFilter, p pagination.Params) ([]CurriculumCourseDetail, pagination.Meta, error)
	UpdateCourse(ctx context.Context, curriculumID, courseID uuid.UUID, req UpdateCurriculumCourseRequest) (*CurriculumCourseDetail, error)
	RemoveCourse(ctx context.Context, curriculumID, courseID uuid.UUID) error
}

type service struct {
	repo   Repository
	logger *slog.Logger
}

// NewService creates a new Service instance for curricula.
func NewService(repo Repository, logger *slog.Logger) Service {
	return &service{
		repo:   repo,
		logger: logger,
	}
}

func (s *service) Create(ctx context.Context, req CreateCurriculumRequest) (*CurriculumResponse, error) {
	// 1. Check parent study program active status
	isActive, err := s.repo.CheckStudyProgramActive(ctx, req.StudyProgramID)
	if err != nil {
		if errors.Is(err, ErrStudyProgramNotFound) {
			return nil, appErrors.ErrNotFound("Program studi tidak ditemukan")
		}
		s.logger.Error("Failed to check study program active", slog.Any("error", err))
		return nil, appErrors.ErrInternal(err)
	}
	if !isActive {
		return nil, appErrors.ErrConflict("Program studi dalam status nonaktif tidak dapat menerima kurikulum baru")
	}

	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		return nil, appErrors.ErrBadRequest("Kode kurikulum tidak boleh kosong")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, appErrors.ErrBadRequest("Nama kurikulum tidak boleh kosong")
	}

	if req.Year < 1970 || req.Year > 2100 {
		return nil, appErrors.ErrBadRequest("Tahun penerapan kurikulum harus berada di antara 1970 dan 2100")
	}

	var desc *string
	if req.Description != nil {
		d := strings.TrimSpace(*req.Description)
		desc = &d
	}

	now := time.Now().UTC()
	c := &Curriculum{
		StudyProgramID: req.StudyProgramID,
		Code:           code,
		Name:           name,
		Year:           req.Year,
		Description:    desc,
		Status:         StatusDraft,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, c); err != nil {
		if errors.Is(err, ErrDuplicateCode) {
			return nil, appErrors.ErrConflict(fmt.Sprintf("Kode kurikulum '%s' sudah digunakan", code))
		}
		if errors.Is(err, ErrStudyProgramNotFound) {
			return nil, appErrors.ErrNotFound("Program studi tidak ditemukan")
		}
		s.logger.Error("Failed to create curriculum", slog.Any("error", err), slog.String("code", code))
		return nil, appErrors.ErrInternal(err)
	}

	stats := &CurriculumStats{TotalCourses: 0, TotalCredits: 0, RequiredCredits: 0, ElectiveCredits: 0}
	s.logger.Info("Curriculum created successfully", slog.String("id", c.ID.String()), slog.String("code", c.Code))
	return s.toResponse(c, stats), nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*CurriculumResponse, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound("Kurikulum tidak ditemukan")
		}
		s.logger.Error("Failed to get curriculum", slog.Any("error", err), slog.String("id", id.String()))
		return nil, appErrors.ErrInternal(err)
	}

	stats, err := s.repo.GetStats(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get curriculum stats", slog.Any("error", err), slog.String("id", id.String()))
		return nil, appErrors.ErrInternal(err)
	}

	return s.toResponse(c, stats), nil
}

func (s *service) List(ctx context.Context, filter ListCurriculaFilter, p pagination.Params) ([]CurriculumResponse, pagination.Meta, error) {
	p.ValidateSort(allowedCurriculaSortColumns)

	list, total, err := s.repo.List(ctx, filter, p)
	if err != nil {
		s.logger.Error("Failed to list curricula", slog.Any("error", err))
		return nil, pagination.Meta{}, appErrors.ErrInternal(err)
	}

	res := make([]CurriculumResponse, 0, len(list))
	for i := range list {
		stats, _ := s.repo.GetStats(ctx, list[i].ID)
		res = append(res, *s.toResponse(&list[i], stats))
	}

	meta := pagination.NewMeta(total, p.Page, p.Limit)
	return res, meta, nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req UpdateCurriculumRequest) (*CurriculumResponse, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound("Kurikulum tidak ditemukan")
		}
		return nil, appErrors.ErrInternal(err)
	}

	if c.Status == StatusArchived {
		return nil, appErrors.ErrConflict("Kurikulum yang telah diarsipkan tidak dapat diubah")
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, appErrors.ErrBadRequest("Nama kurikulum tidak boleh kosong")
		}
		c.Name = name
	}

	if req.Year != nil {
		if *req.Year < 1970 || *req.Year > 2100 {
			return nil, appErrors.ErrBadRequest("Tahun penerapan kurikulum harus berada di antara 1970 dan 2100")
		}
		c.Year = *req.Year
	}

	if req.Description != nil {
		desc := strings.TrimSpace(*req.Description)
		c.Description = &desc
	}

	c.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, c); err != nil {
		s.logger.Error("Failed to update curriculum", slog.Any("error", err), slog.String("id", id.String()))
		return nil, appErrors.ErrInternal(err)
	}

	stats, _ := s.repo.GetStats(ctx, id)
	return s.toResponse(c, stats), nil
}

func (s *service) UpdateStatus(ctx context.Context, id uuid.UUID, req UpdateCurriculumStatusRequest) (*CurriculumResponse, error) {
	targetStatus := strings.ToUpper(strings.TrimSpace(req.Status))
	validStatuses := map[string]bool{
		StatusDraft:    true,
		StatusActive:   true,
		StatusInactive: true,
		StatusArchived: true,
	}
	if !validStatuses[targetStatus] {
		return nil, appErrors.ErrBadRequest(fmt.Sprintf("Status kurikulum '%s' tidak valid. Pilihan: DRAFT, ACTIVE, INACTIVE, ARCHIVED", targetStatus))
	}

	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound("Kurikulum tidak ditemukan")
		}
		return nil, appErrors.ErrInternal(err)
	}

	if c.Status == StatusArchived && targetStatus != StatusArchived {
		return nil, appErrors.ErrConflict("Kurikulum yang telah diarsipkan tidak dapat diaktifkan kembali")
	}

	if err := s.repo.UpdateStatus(ctx, id, targetStatus); err != nil {
		s.logger.Error("Failed to update curriculum status", slog.Any("error", err), slog.String("id", id.String()))
		return nil, appErrors.ErrInternal(err)
	}

	c.Status = targetStatus
	c.UpdatedAt = time.Now().UTC()

	stats, _ := s.repo.GetStats(ctx, id)
	return s.toResponse(c, stats), nil
}

func (s *service) AddCourse(ctx context.Context, curriculumID uuid.UUID, req AddCurriculumCourseRequest) (*CurriculumCourseDetail, error) {
	c, err := s.repo.GetByID(ctx, curriculumID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound("Kurikulum tidak ditemukan")
		}
		return nil, appErrors.ErrInternal(err)
	}

	if c.Status == StatusArchived {
		return nil, appErrors.ErrConflict("Kurikulum yang telah diarsipkan tidak dapat diubah")
	}

	isActiveCourse, err := s.repo.CheckCourseActive(ctx, req.CourseID)
	if err != nil {
		if errors.Is(err, ErrCourseNotFound) {
			return nil, appErrors.ErrNotFound("Mata kuliah tidak ditemukan")
		}
		return nil, appErrors.ErrInternal(err)
	}
	if !isActiveCourse {
		return nil, appErrors.ErrConflict("Mata kuliah nonaktif tidak dapat ditambahkan ke kurikulum")
	}

	if req.Semester < 1 || req.Semester > 14 {
		return nil, appErrors.ErrBadRequest("Semester rekomendasi harus bernilai antara 1 dan 14")
	}

	cat := strings.ToUpper(strings.TrimSpace(req.Category))
	if cat != CategoryRequired && cat != CategoryElective {
		return nil, appErrors.ErrBadRequest("Kategori mata kuliah harus REQUIRED atau ELECTIVE")
	}

	now := time.Now().UTC()
	cc := &CurriculumCourse{
		CurriculumID: curriculumID,
		CourseID:     req.CourseID,
		Semester:     req.Semester,
		Category:     cat,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.AddCourse(ctx, cc); err != nil {
		if errors.Is(err, ErrDuplicateCourse) {
			return nil, appErrors.ErrConflict("Mata kuliah sudah terdaftar dalam kurikulum ini")
		}
		if errors.Is(err, ErrCourseNotFound) {
			return nil, appErrors.ErrNotFound("Mata kuliah tidak ditemukan")
		}
		s.logger.Error("Failed to add course to curriculum", slog.Any("error", err))
		return nil, appErrors.ErrInternal(err)
	}

	return s.repo.GetCourse(ctx, curriculumID, req.CourseID)
}

func (s *service) GetCourse(ctx context.Context, curriculumID, courseID uuid.UUID) (*CurriculumCourseDetail, error) {
	detail, err := s.repo.GetCourse(ctx, curriculumID, courseID)
	if err != nil {
		if errors.Is(err, ErrCourseNotFound) {
			return nil, appErrors.ErrNotFound("Mata kuliah tidak ditemukan dalam kurikulum ini")
		}
		return nil, appErrors.ErrInternal(err)
	}
	return detail, nil
}

func (s *service) ListCourses(ctx context.Context, curriculumID uuid.UUID, filter ListCurriculumCoursesFilter, p pagination.Params) ([]CurriculumCourseDetail, pagination.Meta, error) {
	// Verify curriculum exists
	if _, err := s.repo.GetByID(ctx, curriculumID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, pagination.Meta{}, appErrors.ErrNotFound("Kurikulum tidak ditemukan")
		}
		return nil, pagination.Meta{}, appErrors.ErrInternal(err)
	}

	p.ValidateSort(allowedCurriculumCoursesSortColumns)

	list, total, err := s.repo.ListCourses(ctx, curriculumID, filter, p)
	if err != nil {
		s.logger.Error("Failed to list curriculum courses", slog.Any("error", err))
		return nil, pagination.Meta{}, appErrors.ErrInternal(err)
	}

	meta := pagination.NewMeta(total, p.Page, p.Limit)
	return list, meta, nil
}

func (s *service) UpdateCourse(ctx context.Context, curriculumID, courseID uuid.UUID, req UpdateCurriculumCourseRequest) (*CurriculumCourseDetail, error) {
	c, err := s.repo.GetByID(ctx, curriculumID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, appErrors.ErrNotFound("Kurikulum tidak ditemukan")
		}
		return nil, appErrors.ErrInternal(err)
	}

	if c.Status == StatusArchived {
		return nil, appErrors.ErrConflict("Kurikulum yang telah diarsipkan tidak dapat diubah")
	}

	if req.Semester != nil && (*req.Semester < 1 || *req.Semester > 14) {
		return nil, appErrors.ErrBadRequest("Semester rekomendasi harus bernilai antara 1 dan 14")
	}

	if req.Category != nil {
		cat := strings.ToUpper(strings.TrimSpace(*req.Category))
		if cat != CategoryRequired && cat != CategoryElective {
			return nil, appErrors.ErrBadRequest("Kategori mata kuliah harus REQUIRED atau ELECTIVE")
		}
		req.Category = &cat
	}

	if err := s.repo.UpdateCourse(ctx, curriculumID, courseID, req.Semester, req.Category); err != nil {
		if errors.Is(err, ErrCourseNotFound) {
			return nil, appErrors.ErrNotFound("Mata kuliah tidak ditemukan dalam kurikulum ini")
		}
		return nil, appErrors.ErrInternal(err)
	}

	return s.repo.GetCourse(ctx, curriculumID, courseID)
}

func (s *service) RemoveCourse(ctx context.Context, curriculumID, courseID uuid.UUID) error {
	c, err := s.repo.GetByID(ctx, curriculumID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return appErrors.ErrNotFound("Kurikulum tidak ditemukan")
		}
		return appErrors.ErrInternal(err)
	}

	if c.Status == StatusArchived {
		return appErrors.ErrConflict("Kurikulum yang telah diarsipkan tidak dapat diubah")
	}

	if err := s.repo.RemoveCourse(ctx, curriculumID, courseID); err != nil {
		if errors.Is(err, ErrCourseNotFound) {
			return appErrors.ErrNotFound("Mata kuliah tidak ditemukan dalam kurikulum ini")
		}
		s.logger.Error("Failed to remove course from curriculum", slog.Any("error", err))
		return appErrors.ErrInternal(err)
	}

	s.logger.Info("Course removed from curriculum",
		slog.String("curriculum_id", curriculumID.String()),
		slog.String("course_id", courseID.String()))
	return nil
}

func (s *service) toResponse(c *Curriculum, stats *CurriculumStats) *CurriculumResponse {
	return &CurriculumResponse{
		ID:             c.ID,
		StudyProgramID: c.StudyProgramID,
		Code:           c.Code,
		Name:           c.Name,
		Year:           c.Year,
		Description:    c.Description,
		Status:         c.Status,
		Stats:          stats,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}
