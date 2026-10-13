package classes

import (
	"time"

	"github.com/google/uuid"
)

type Class struct {
	ID               uuid.UUID `json:"id"`
	CourseOfferingID uuid.UUID `json:"course_offering_id"`
	Code             string    `json:"code"`
	Name             string    `json:"name"`
	Capacity         int       `json:"capacity"`
	Status           string    `json:"status"` // DRAFT, OPEN, IN_PROGRESS, CLOSED, CANCELLED
	Remarks          *string   `json:"remarks,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ClassLecturer struct {
	ID         uuid.UUID `json:"id"`
	ClassID    uuid.UUID `json:"class_id"`
	LecturerID uuid.UUID `json:"lecturer_id"`
	Role       string    `json:"role"`   // COORDINATOR, TEAM_TEACHING
	Status     string    `json:"status"` // ACTIVE, INACTIVE
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ClassLecturerDetail struct {
	ClassLecturer
	LecturerNIDN  string `json:"lecturer_nidn"`
	LecturerName  string `json:"lecturer_name"`
	LecturerEmail string `json:"lecturer_email"`
}

type ClassSchedule struct {
	ID           uuid.UUID `json:"id"`
	ClassID      uuid.UUID `json:"class_id"`
	DayOfWeek    int       `json:"day_of_week"` // 1 = Monday ... 7 = Sunday
	StartTime    string    `json:"start_time"`  // Format: HH:MM:SS or HH:MM
	EndTime      string    `json:"end_time"`    // Format: HH:MM:SS or HH:MM
	Room         *string   `json:"room,omitempty"`
	DeliveryMode string    `json:"delivery_mode"` // OFFLINE, ONLINE, HYBRID
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ClassDetail struct {
	Class
	CourseCode         string                `json:"course_code"`
	CourseName         string                `json:"course_name"`
	CourseCredits      int                   `json:"course_credits"`
	AcademicPeriodID   uuid.UUID             `json:"academic_period_id"`
	AcademicPeriodCode string                `json:"academic_period_code"`
	AcademicPeriodName string                `json:"academic_period_name"`
	StudyProgramID     uuid.UUID             `json:"study_program_id"`
	StudyProgramName   string                `json:"study_program_name"`
	Lecturers          []ClassLecturerDetail `json:"lecturers"`
	Schedules          []ClassSchedule       `json:"schedules"`
}
