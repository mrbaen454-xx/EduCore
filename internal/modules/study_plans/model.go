package study_plans

import (
	"time"

	"educore/internal/modules/classes"

	"github.com/google/uuid"
)

// StudyPlan represents a student's study plan (KRS) for a specific academic period.
type StudyPlan struct {
	ID               uuid.UUID `json:"id"`
	StudentID        uuid.UUID `json:"student_id"`
	AcademicPeriodID uuid.UUID `json:"academic_period_id"`
	TotalCredits     int       `json:"total_credits"`
	Status           string    `json:"status"` // DRAFT, ACTIVE, CANCELLED
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Enrollment represents a course class registration item within a study plan.
type Enrollment struct {
	ID          uuid.UUID  `json:"id"`
	StudyPlanID uuid.UUID  `json:"study_plan_id"`
	StudentID   uuid.UUID  `json:"student_id"`
	ClassID     uuid.UUID  `json:"class_id"`
	Status      string     `json:"status"` // ENROLLED, DROPPED, CANCELLED
	EnrolledAt  time.Time  `json:"enrolled_at"`
	DroppedAt   *time.Time `json:"dropped_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// EnrollmentDetail represents an enrollment with full course, class, lecturer, and schedule metadata.
type EnrollmentDetail struct {
	Enrollment
	ClassCode     string                        `json:"class_code"`
	ClassName     string                        `json:"class_name"`
	CourseID      uuid.UUID                     `json:"course_id"`
	CourseCode    string                        `json:"course_code"`
	CourseName    string                        `json:"course_name"`
	CourseCredits int                           `json:"course_credits"`
	Lecturers     []classes.ClassLecturerDetail `json:"lecturers"`
	Schedules     []classes.ClassSchedule       `json:"schedules"`
}

// StudyPlanDetail represents a complete study plan with student, academic period, and enrollment details.
type StudyPlanDetail struct {
	StudyPlan
	StudentNIM         string             `json:"student_nim"`
	StudentName        string             `json:"student_name"`
	StudyProgramID     uuid.UUID          `json:"study_program_id"`
	StudyProgramCode   string             `json:"study_program_code"`
	StudyProgramName   string             `json:"study_program_name"`
	AcademicPeriodCode string             `json:"academic_period_code"`
	AcademicPeriodName string             `json:"academic_period_name"`
	Enrollments        []EnrollmentDetail `json:"enrollments"`
}

// AvailableClass represents a class offering available for student enrollment.
type AvailableClass struct {
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
