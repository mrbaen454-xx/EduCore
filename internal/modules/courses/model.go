package courses

import (
	"time"

	"github.com/google/uuid"
)

// Course represents the domain model for an academic course catalog item.
type Course struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Credits     int       `json:"credits"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CoursePrerequisite represents the relationship between a course and its prerequisite course.
type CoursePrerequisite struct {
	ID                   uuid.UUID `json:"id"`
	CourseID             uuid.UUID `json:"course_id"`
	PrerequisiteCourseID uuid.UUID `json:"prerequisite_course_id"`
	CreatedAt            time.Time `json:"created_at"`
}

// PrerequisiteDetail represents summary information of a prerequisite course.
type PrerequisiteDetail struct {
	ID      uuid.UUID `json:"id"`
	Code    string    `json:"code"`
	Name    string    `json:"name"`
	Credits int       `json:"credits"`
}
