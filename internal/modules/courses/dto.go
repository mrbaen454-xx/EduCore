package courses

import (
	"time"

	"github.com/google/uuid"
)

// CreateCourseRequest represents payload to create a new course.
type CreateCourseRequest struct {
	Code        string  `json:"code" validate:"required,min=2,max=20"`
	Name        string  `json:"name" validate:"required,min=1,max=150"`
	Description *string `json:"description,omitempty"`
	Credits     int     `json:"credits" validate:"required,min=1,max=8"`
}

// UpdateCourseRequest represents payload to update existing course attributes.
type UpdateCourseRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=150"`
	Description *string `json:"description,omitempty"`
	Credits     *int    `json:"credits,omitempty" validate:"omitempty,min=1,max=8"`
}

// UpdateCourseStatusRequest represents payload to activate/deactivate a course.
type UpdateCourseStatusRequest struct {
	IsActive *bool `json:"is_active" validate:"required"`
}

// AddPrerequisiteRequest represents payload to link a prerequisite course.
type AddPrerequisiteRequest struct {
	PrerequisiteCourseID uuid.UUID `json:"prerequisite_course_id" validate:"required"`
}

// CourseResponse represents standard API output for a course entity.
type CourseResponse struct {
	ID            uuid.UUID            `json:"id"`
	Code          string               `json:"code"`
	Name          string               `json:"name"`
	Description   *string              `json:"description,omitempty"`
	Credits       int                  `json:"credits"`
	IsActive      bool                 `json:"is_active"`
	Prerequisites []PrerequisiteDetail `json:"prerequisites,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

// ListCoursesFilter contains query parameters for filtering course catalogs.
type ListCoursesFilter struct {
	Search   string
	Credits  *int
	IsActive *bool
}
