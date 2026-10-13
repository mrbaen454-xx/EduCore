package course_offerings

import (
	"time"

	"github.com/google/uuid"
)

type CourseOffering struct {
	ID               uuid.UUID  `json:"id"`
	AcademicPeriodID uuid.UUID  `json:"academic_period_id"`
	CourseID         uuid.UUID  `json:"course_id"`
	CurriculumID     *uuid.UUID `json:"curriculum_id,omitempty"`
	StudyProgramID   uuid.UUID  `json:"study_program_id"`
	Status           string     `json:"status"` // PLANNED, OPEN, CLOSED, CANCELLED
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type CourseOfferingDetail struct {
	CourseOffering
	AcademicPeriodCode string `json:"academic_period_code"`
	AcademicPeriodName string `json:"academic_period_name"`
	CourseCode         string `json:"course_code"`
	CourseName         string `json:"course_name"`
	CourseCredits      int    `json:"course_credits"`
	StudyProgramCode   string `json:"study_program_code"`
	StudyProgramName   string `json:"study_program_name"`
	CurriculumName     string `json:"curriculum_name,omitempty"`
	TotalClasses       int    `json:"total_classes"`
}
