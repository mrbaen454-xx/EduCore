package course_offerings

import (
	"time"

	"github.com/google/uuid"
)

type CreateCourseOfferingRequest struct {
	AcademicPeriodID uuid.UUID  `json:"academic_period_id" binding:"required"`
	CourseID         uuid.UUID  `json:"course_id" binding:"required"`
	CurriculumID     *uuid.UUID `json:"curriculum_id"`
	StudyProgramID   uuid.UUID  `json:"study_program_id" binding:"required"`
}

type UpdateCourseOfferingStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=PLANNED OPEN CLOSED CANCELLED"`
}

type CourseOfferingFilter struct {
	AcademicPeriodID *uuid.UUID
	StudyProgramID   *uuid.UUID
	CurriculumID     *uuid.UUID
	CourseID         *uuid.UUID
	Status           string
	Search           string
}

type CourseOfferingResponse struct {
	ID                 uuid.UUID  `json:"id"`
	AcademicPeriodID   uuid.UUID  `json:"academic_period_id"`
	AcademicPeriodCode string     `json:"academic_period_code"`
	AcademicPeriodName string     `json:"academic_period_name"`
	CourseID           uuid.UUID  `json:"course_id"`
	CourseCode         string     `json:"course_code"`
	CourseName         string     `json:"course_name"`
	CourseCredits      int        `json:"course_credits"`
	CurriculumID       *uuid.UUID `json:"curriculum_id,omitempty"`
	CurriculumName     string     `json:"curriculum_name,omitempty"`
	StudyProgramID     uuid.UUID  `json:"study_program_id"`
	StudyProgramCode   string     `json:"study_program_code"`
	StudyProgramName   string     `json:"study_program_name"`
	Status             string     `json:"status"`
	TotalClasses       int        `json:"total_classes"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func ToResponse(d *CourseOfferingDetail) CourseOfferingResponse {
	return CourseOfferingResponse{
		ID:                 d.ID,
		AcademicPeriodID:   d.AcademicPeriodID,
		AcademicPeriodCode: d.AcademicPeriodCode,
		AcademicPeriodName: d.AcademicPeriodName,
		CourseID:           d.CourseID,
		CourseCode:         d.CourseCode,
		CourseName:         d.CourseName,
		CourseCredits:      d.CourseCredits,
		CurriculumID:       d.CurriculumID,
		CurriculumName:     d.CurriculumName,
		StudyProgramID:     d.StudyProgramID,
		StudyProgramCode:   d.StudyProgramCode,
		StudyProgramName:   d.StudyProgramName,
		Status:             d.Status,
		TotalClasses:       d.TotalClasses,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
	}
}
