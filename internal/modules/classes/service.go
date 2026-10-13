package classes

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
)

type Service interface {
	// Classes
	CreateClass(ctx context.Context, req CreateClassRequest) (*ClassDetailResponse, error)
	GetClassByID(ctx context.Context, id uuid.UUID) (*ClassDetailResponse, error)
	ListClasses(ctx context.Context, filter ClassFilter, p pagination.Params) ([]ClassResponse, int, error)
	UpdateClass(ctx context.Context, id uuid.UUID, req UpdateClassRequest) (*ClassDetailResponse, error)
	UpdateClassStatus(ctx context.Context, id uuid.UUID, req UpdateClassStatusRequest) error

	// Lecturers
	AssignLecturer(ctx context.Context, classID uuid.UUID, req AssignLecturerRequest) (*ClassDetailResponse, error)
	RemoveLecturer(ctx context.Context, classID, lecturerID uuid.UUID) (*ClassDetailResponse, error)

	// Schedules
	CreateSchedule(ctx context.Context, classID uuid.UUID, req CreateScheduleRequest) (*ClassSchedule, error)
	UpdateSchedule(ctx context.Context, classID, scheduleID uuid.UUID, req UpdateScheduleRequest) (*ClassSchedule, error)
	DeleteSchedule(ctx context.Context, classID, scheduleID uuid.UUID) error
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

func (s *service) CreateClass(ctx context.Context, req CreateClassRequest) (*ClassDetailResponse, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	name := strings.TrimSpace(req.Name)

	if code == "" || name == "" {
		return nil, appErrors.ErrBadRequest("Kode dan nama kelas wajib diisi")
	}
	if req.Capacity <= 0 {
		return nil, appErrors.ErrBadRequest("Kapasitas kelas harus lebih besar dari 0")
	}

	c := &Class{
		CourseOfferingID: req.CourseOfferingID,
		Code:             code,
		Name:             name,
		Capacity:         req.Capacity,
		Status:           "DRAFT",
		Remarks:          req.Remarks,
	}

	if err := s.repo.CreateClass(ctx, c); err != nil {
		return nil, err
	}

	detail, err := s.repo.GetClassByID(ctx, c.ID)
	if err != nil {
		return nil, err
	}

	resp := ToDetailResponse(detail)
	return &resp, nil
}

func (s *service) GetClassByID(ctx context.Context, id uuid.UUID) (*ClassDetailResponse, error) {
	detail, err := s.repo.GetClassByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := ToDetailResponse(detail)
	return &resp, nil
}

func (s *service) ListClasses(ctx context.Context, filter ClassFilter, p pagination.Params) ([]ClassResponse, int, error) {
	list, total, err := s.repo.ListClasses(ctx, filter, p)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]ClassResponse, len(list))
	for i, item := range list {
		responses[i] = ToResponse(&item)
	}

	return responses, total, nil
}

func (s *service) UpdateClass(ctx context.Context, id uuid.UUID, req UpdateClassRequest) (*ClassDetailResponse, error) {
	detail, err := s.repo.GetClassByID(ctx, id)
	if err != nil {
		return nil, err
	}

	cls := &detail.Class

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, appErrors.ErrBadRequest("Nama kelas tidak boleh kosong")
		}
		cls.Name = name
	}
	if req.Capacity != nil {
		if *req.Capacity <= 0 {
			return nil, appErrors.ErrBadRequest("Kapasitas kelas harus lebih besar dari 0")
		}
		cls.Capacity = *req.Capacity
	}
	if req.Remarks != nil {
		cls.Remarks = req.Remarks
	}

	if err := s.repo.UpdateClass(ctx, cls); err != nil {
		return nil, err
	}

	updatedDetail, err := s.repo.GetClassByID(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := ToDetailResponse(updatedDetail)
	return &resp, nil
}

func (s *service) UpdateClassStatus(ctx context.Context, id uuid.UUID, req UpdateClassStatusRequest) error {
	return s.repo.UpdateClassStatus(ctx, id, req.Status)
}

func (s *service) AssignLecturer(ctx context.Context, classID uuid.UUID, req AssignLecturerRequest) (*ClassDetailResponse, error) {
	if _, err := s.repo.GetClassByID(ctx, classID); err != nil {
		return nil, err
	}

	cl := &ClassLecturer{
		ClassID:    classID,
		LecturerID: req.LecturerID,
		Role:       req.Role,
		Status:     "ACTIVE",
	}

	if err := s.repo.AssignLecturer(ctx, cl); err != nil {
		return nil, err
	}

	// Re-fetch detail
	updatedDetail, err := s.repo.GetClassByID(ctx, classID)
	if err != nil {
		return nil, err
	}

	resp := ToDetailResponse(updatedDetail)
	return &resp, nil
}

func (s *service) RemoveLecturer(ctx context.Context, classID, lecturerID uuid.UUID) (*ClassDetailResponse, error) {
	if err := s.repo.RemoveLecturer(ctx, classID, lecturerID); err != nil {
		return nil, err
	}

	updatedDetail, err := s.repo.GetClassByID(ctx, classID)
	if err != nil {
		return nil, err
	}

	resp := ToDetailResponse(updatedDetail)
	return &resp, nil
}

func parseTime(tStr string) (time.Time, error) {
	tStr = strings.TrimSpace(tStr)
	// Try HH:MM:SS
	if t, err := time.Parse("15:04:05", tStr); err == nil {
		return t, nil
	}
	// Try HH:MM
	if t, err := time.Parse("15:04", tStr); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("format waktu tidak valid (gunakan HH:MM atau HH:MM:SS)")
}

func (s *service) CreateSchedule(ctx context.Context, classID uuid.UUID, req CreateScheduleRequest) (*ClassSchedule, error) {
	detail, err := s.repo.GetClassByID(ctx, classID)
	if err != nil {
		return nil, err
	}

	st, err := parseTime(req.StartTime)
	if err != nil {
		return nil, appErrors.ErrBadRequest(err.Error())
	}
	et, err := parseTime(req.EndTime)
	if err != nil {
		return nil, appErrors.ErrBadRequest(err.Error())
	}

	if !st.Before(et) {
		return nil, appErrors.ErrBadRequest("Waktu mulai harus lebih awal dari waktu selesai")
	}

	normalizedStart := st.Format("15:04:05")
	normalizedEnd := et.Format("15:04:05")

	// 1. Check self class schedule conflict
	selfConflict, err := s.repo.CheckClassSelfConflict(ctx, classID, uuid.Nil, req.DayOfWeek, normalizedStart, normalizedEnd)
	if err != nil {
		return nil, err
	}
	if selfConflict {
		return nil, appErrors.ErrConflict("Kelas sudah memiliki jadwal lain pada interval waktu dan hari yang sama")
	}

	// 2. Check lecturer conflict
	lecturerConflict, reason, err := s.repo.CheckLecturerScheduleConflict(ctx, detail.AcademicPeriodID, classID, uuid.Nil, req.DayOfWeek, normalizedStart, normalizedEnd)
	if err != nil {
		return nil, err
	}
	if lecturerConflict {
		return nil, appErrors.ErrConflict(reason)
	}

	// 3. Check room conflict
	var roomStr string
	if req.Room != nil {
		roomStr = strings.TrimSpace(*req.Room)
	}
	if roomStr != "" {
		roomConflict, reason, err := s.repo.CheckRoomScheduleConflict(ctx, detail.AcademicPeriodID, classID, uuid.Nil, req.DayOfWeek, normalizedStart, normalizedEnd, roomStr)
		if err != nil {
			return nil, err
		}
		if roomConflict {
			return nil, appErrors.ErrConflict(reason)
		}
	}

	cs := &ClassSchedule{
		ClassID:      classID,
		DayOfWeek:    req.DayOfWeek,
		StartTime:    normalizedStart,
		EndTime:      normalizedEnd,
		Room:         req.Room,
		DeliveryMode: req.DeliveryMode,
	}

	if err := s.repo.CreateSchedule(ctx, cs); err != nil {
		return nil, err
	}

	return cs, nil
}

func (s *service) UpdateSchedule(ctx context.Context, classID, scheduleID uuid.UUID, req UpdateScheduleRequest) (*ClassSchedule, error) {
	detail, err := s.repo.GetClassByID(ctx, classID)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.GetScheduleByID(ctx, scheduleID)
	if err != nil {
		return nil, err
	}
	if existing.ClassID != classID {
		return nil, appErrors.ErrNotFound("Jadwal tidak ditemukan pada kelas ini")
	}

	dayOfWeek := existing.DayOfWeek
	if req.DayOfWeek != nil {
		dayOfWeek = *req.DayOfWeek
	}

	startTime := existing.StartTime
	if req.StartTime != nil {
		st, err := parseTime(*req.StartTime)
		if err != nil {
			return nil, appErrors.ErrBadRequest(err.Error())
		}
		startTime = st.Format("15:04:05")
	}

	endTime := existing.EndTime
	if req.EndTime != nil {
		et, err := parseTime(*req.EndTime)
		if err != nil {
			return nil, appErrors.ErrBadRequest(err.Error())
		}
		endTime = et.Format("15:04:05")
	}

	st, _ := parseTime(startTime)
	et, _ := parseTime(endTime)
	if !st.Before(et) {
		return nil, appErrors.ErrBadRequest("Waktu mulai harus lebih awal dari waktu selesai")
	}

	// Check self conflict excluding current schedule
	selfConflict, err := s.repo.CheckClassSelfConflict(ctx, classID, scheduleID, dayOfWeek, startTime, endTime)
	if err != nil {
		return nil, err
	}
	if selfConflict {
		return nil, appErrors.ErrConflict("Kelas sudah memiliki jadwal lain pada interval waktu dan hari yang sama")
	}

	// Check lecturer conflict
	lecturerConflict, reason, err := s.repo.CheckLecturerScheduleConflict(ctx, detail.AcademicPeriodID, classID, scheduleID, dayOfWeek, startTime, endTime)
	if err != nil {
		return nil, err
	}
	if lecturerConflict {
		return nil, appErrors.ErrConflict(reason)
	}

	// Check room conflict
	room := existing.Room
	if req.Room != nil {
		room = req.Room
	}
	var roomStr string
	if room != nil {
		roomStr = strings.TrimSpace(*room)
	}
	if roomStr != "" {
		roomConflict, reason, err := s.repo.CheckRoomScheduleConflict(ctx, detail.AcademicPeriodID, classID, scheduleID, dayOfWeek, startTime, endTime, roomStr)
		if err != nil {
			return nil, err
		}
		if roomConflict {
			return nil, appErrors.ErrConflict(reason)
		}
	}

	existing.DayOfWeek = dayOfWeek
	existing.StartTime = startTime
	existing.EndTime = endTime
	existing.Room = room
	if req.DeliveryMode != nil {
		existing.DeliveryMode = *req.DeliveryMode
	}

	if err := s.repo.UpdateSchedule(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (s *service) DeleteSchedule(ctx context.Context, classID, scheduleID uuid.UUID) error {
	existing, err := s.repo.GetScheduleByID(ctx, scheduleID)
	if err != nil {
		return err
	}
	if existing.ClassID != classID {
		return appErrors.ErrNotFound("Jadwal tidak ditemukan pada kelas ini")
	}

	return s.repo.DeleteSchedule(ctx, scheduleID)
}
