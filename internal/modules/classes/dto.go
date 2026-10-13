package classes

import (
	"time"

	"github.com/google/uuid"
)

type CreateClassRequest struct {
	CourseOfferingID uuid.UUID `json:"course_offering_id" binding:"required"`
	Code             string    `json:"code" binding:"required,min=1,max=20"`
	Name             string    `json:"name" binding:"required,min=1,max=100"`
	Capacity         int       `json:"capacity" binding:"required,min=1"`
	Remarks          *string   `json:"remarks"`
}

type UpdateClassRequest struct {
	Name     *string `json:"name" binding:"omitempty,min=1,max=100"`
	Capacity *int    `json:"capacity" binding:"omitempty,min=1"`
	Remarks  *string `json:"remarks"`
}

type UpdateClassStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=DRAFT OPEN IN_PROGRESS CLOSED CANCELLED"`
}

type AssignLecturerRequest struct {
	LecturerID uuid.UUID `json:"lecturer_id" binding:"required"`
	Role       string    `json:"role" binding:"required,oneof=COORDINATOR TEAM_TEACHING"`
}

type CreateScheduleRequest struct {
	DayOfWeek    int     `json:"day_of_week" binding:"required,min=1,max=7"`
	StartTime    string  `json:"start_time" binding:"required"` // Format: HH:MM or HH:MM:SS
	EndTime      string  `json:"end_time" binding:"required"`   // Format: HH:MM or HH:MM:SS
	Room         *string `json:"room"`
	DeliveryMode string  `json:"delivery_mode" binding:"required,oneof=OFFLINE ONLINE HYBRID"`
}

type UpdateScheduleRequest struct {
	DayOfWeek    *int    `json:"day_of_week" binding:"omitempty,min=1,max=7"`
	StartTime    *string `json:"start_time"`
	EndTime      *string `json:"end_time"`
	Room         *string `json:"room"`
	DeliveryMode *string `json:"delivery_mode" binding:"omitempty,oneof=OFFLINE ONLINE HYBRID"`
}

type ClassFilter struct {
	AcademicPeriodID *uuid.UUID
	CourseOfferingID *uuid.UUID
	StudyProgramID   *uuid.UUID
	Status           string
	Search           string
}

type ClassResponse struct {
	ID                 uuid.UUID `json:"id"`
	CourseOfferingID   uuid.UUID `json:"course_offering_id"`
	Code               string    `json:"code"`
	Name               string    `json:"name"`
	Capacity           int       `json:"capacity"`
	Status             string    `json:"status"`
	Remarks            *string   `json:"remarks,omitempty"`
	CourseCode         string    `json:"course_code"`
	CourseName         string    `json:"course_name"`
	CourseCredits      int       `json:"course_credits"`
	AcademicPeriodCode string    `json:"academic_period_code"`
	StudyProgramName   string    `json:"study_program_name"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ClassDetailResponse struct {
	ClassResponse
	AcademicPeriodID   uuid.UUID             `json:"academic_period_id"`
	AcademicPeriodName string                `json:"academic_period_name"`
	StudyProgramID     uuid.UUID             `json:"study_program_id"`
	Lecturers          []ClassLecturerDetail `json:"lecturers"`
	Schedules          []ClassSchedule       `json:"schedules"`
}

func ToResponse(d *ClassDetail) ClassResponse {
	return ClassResponse{
		ID:                 d.ID,
		CourseOfferingID:   d.CourseOfferingID,
		Code:               d.Code,
		Name:               d.Name,
		Capacity:           d.Capacity,
		Status:             d.Status,
		Remarks:            d.Remarks,
		CourseCode:         d.CourseCode,
		CourseName:         d.CourseName,
		CourseCredits:      d.CourseCredits,
		AcademicPeriodCode: d.AcademicPeriodCode,
		StudyProgramName:   d.StudyProgramName,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
	}
}

func ToDetailResponse(d *ClassDetail) ClassDetailResponse {
	return ClassDetailResponse{
		ClassResponse:      ToResponse(d),
		AcademicPeriodID:   d.AcademicPeriodID,
		AcademicPeriodName: d.AcademicPeriodName,
		StudyProgramID:     d.StudyProgramID,
		Lecturers:          d.Lecturers,
		Schedules:          d.Schedules,
	}
}
