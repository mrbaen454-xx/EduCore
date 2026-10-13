package study_plans

import (
	"educore/internal/modules/classes"

	"github.com/google/uuid"
)

// CreateStudyPlanRequest represents input for creating or retrieving a study plan.
type CreateStudyPlanRequest struct {
	AcademicPeriodID uuid.UUID  `json:"academic_period_id" binding:"required"`
	StudentID        *uuid.UUID `json:"student_id,omitempty"` // For administrative endpoints only
}

// EnrollClassRequest represents input for registering a class into a study plan.
type EnrollClassRequest struct {
	ClassID uuid.UUID `json:"class_id" binding:"required"`
}

// StudyPlanFilter holds filtering parameters for querying study plans.
type StudyPlanFilter struct {
	StudentID        *uuid.UUID
	AcademicPeriodID *uuid.UUID
	Status           string
}

// AvailableClassFilter holds filtering parameters for browsing available classes.
type AvailableClassFilter struct {
	AcademicPeriodID *uuid.UUID
	StudyProgramID   *uuid.UUID
	Search           string
}

// StudyPlanResponse is the API response payload for basic study plan information.
type StudyPlanResponse struct {
	ID                 uuid.UUID `json:"id"`
	StudentID          uuid.UUID `json:"student_id"`
	StudentNIM         string    `json:"student_nim,omitempty"`
	StudentName        string    `json:"student_name,omitempty"`
	StudyProgramID     uuid.UUID `json:"study_program_id,omitempty"`
	StudyProgramName   string    `json:"study_program_name,omitempty"`
	AcademicPeriodID   uuid.UUID `json:"academic_period_id"`
	AcademicPeriodCode string    `json:"academic_period_code,omitempty"`
	AcademicPeriodName string    `json:"academic_period_name,omitempty"`
	TotalCredits       int       `json:"total_credits"`
	Status             string    `json:"status"`
	CreatedAt          string    `json:"created_at"`
	UpdatedAt          string    `json:"updated_at"`
}

// EnrollmentResponse is the API response payload for an enrolled class.
type EnrollmentResponse struct {
	ID            uuid.UUID                     `json:"id"`
	StudyPlanID   uuid.UUID                     `json:"study_plan_id"`
	StudentID     uuid.UUID                     `json:"student_id"`
	ClassID       uuid.UUID                     `json:"class_id"`
	ClassCode     string                        `json:"class_code"`
	ClassName     string                        `json:"class_name"`
	CourseID      uuid.UUID                     `json:"course_id"`
	CourseCode    string                        `json:"course_code"`
	CourseName    string                        `json:"course_name"`
	CourseCredits int                           `json:"course_credits"`
	Status        string                        `json:"status"`
	EnrolledAt    string                        `json:"enrolled_at"`
	DroppedAt     *string                       `json:"dropped_at,omitempty"`
	Lecturers     []classes.ClassLecturerDetail `json:"lecturers,omitempty"`
	Schedules     []classes.ClassSchedule       `json:"schedules,omitempty"`
}

// StudyPlanDetailResponse is the API response payload for complete study plan details.
type StudyPlanDetailResponse struct {
	StudyPlanResponse
	Enrollments []EnrollmentResponse `json:"enrollments"`
}

// AvailableClassResponse is the API response payload for browsing classes available for KRS.
type AvailableClassResponse struct {
	ClassID            uuid.UUID                     `json:"class_id"`
	ClassCode          string                        `json:"class_code"`
	ClassName          string                        `json:"class_name"`
	Capacity           int                           `json:"capacity"`
	EnrolledCount      int                           `json:"enrolled_count"`
	AvailableSeats     int                           `json:"available_seats"`
	ClassStatus        string                        `json:"class_status"`
	CourseOfferingID   uuid.UUID                     `json:"course_offering_id"`
	OfferingStatus     string                        `json:"offering_status"`
	CourseID           uuid.UUID                     `json:"course_id"`
	CourseCode         string                        `json:"course_code"`
	CourseName         string                        `json:"course_name"`
	CourseCredits      int                           `json:"course_credits"`
	AcademicPeriodID   uuid.UUID                     `json:"academic_period_id"`
	AcademicPeriodCode string                        `json:"academic_period_code"`
	StudyProgramID     uuid.UUID                     `json:"study_program_id"`
	StudyProgramName   string                        `json:"study_program_name"`
	Lecturers          []classes.ClassLecturerDetail `json:"lecturers"`
	Schedules          []classes.ClassSchedule       `json:"schedules"`
}
