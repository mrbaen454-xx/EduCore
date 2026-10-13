package course_offerings

import (
	"context"
	"log/slog"

	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
)

type Service interface {
	Create(ctx context.Context, req CreateCourseOfferingRequest) (*CourseOfferingResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*CourseOfferingResponse, error)
	List(ctx context.Context, filter CourseOfferingFilter, p pagination.Params) ([]CourseOfferingResponse, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, req UpdateCourseOfferingStatusRequest) error
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

func (s *service) Create(ctx context.Context, req CreateCourseOfferingRequest) (*CourseOfferingResponse, error) {
	_, courseActive, prodiActive, _, err := s.repo.CheckForeignEntities(
		ctx, req.AcademicPeriodID, req.CourseID, req.StudyProgramID, req.CurriculumID,
	)
	if err != nil {
		return nil, err
	}

	if !courseActive {
		return nil, appErrors.ErrConflict("Mata kuliah nonaktif tidak dapat ditawarkan")
	}

	if !prodiActive {
		return nil, appErrors.ErrConflict("Program studi nonaktif tidak dapat membuat penawaran")
	}

	co := &CourseOffering{
		AcademicPeriodID: req.AcademicPeriodID,
		CourseID:         req.CourseID,
		CurriculumID:     req.CurriculumID,
		StudyProgramID:   req.StudyProgramID,
		Status:           "PLANNED",
	}

	if err := s.repo.Create(ctx, co); err != nil {
		return nil, err
	}

	// Fetch detail
	detail, err := s.repo.GetByID(ctx, co.ID)
	if err != nil {
		return nil, err
	}

	resp := ToResponse(detail)
	return &resp, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*CourseOfferingResponse, error) {
	detail, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := ToResponse(detail)
	return &resp, nil
}

func (s *service) List(ctx context.Context, filter CourseOfferingFilter, p pagination.Params) ([]CourseOfferingResponse, int, error) {
	list, total, err := s.repo.List(ctx, filter, p)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]CourseOfferingResponse, len(list))
	for i, item := range list {
		responses[i] = ToResponse(&item)
	}

	return responses, total, nil
}

func (s *service) UpdateStatus(ctx context.Context, id uuid.UUID, req UpdateCourseOfferingStatusRequest) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.Status == "CANCELLED" && req.Status != "CANCELLED" {
		return appErrors.ErrConflict("Penawaran yang telah dibatalkan tidak dapat diaktifkan kembali")
	}

	return s.repo.UpdateStatus(ctx, id, req.Status)
}
