package curricula

import (
	"time"

	"github.com/google/uuid"
)

// CreateCurriculumRequest represents payload to create a new curriculum.
type CreateCurriculumRequest struct {
	StudyProgramID uuid.UUID `json:"study_program_id" validate:"required"`
	Code           string    `json:"code" validate:"required,min=2,max=30"`
	Name           string    `json:"name" validate:"required,min=1,max=150"`
	Year           int       `json:"year" validate:"required,min=1970,max=2100"`
	Description    *string   `json:"description,omitempty"`
}

// UpdateCurriculumRequest represents payload to update curriculum metadata.
type UpdateCurriculumRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=150"`
	Year        *int    `json:"year,omitempty" validate:"omitempty,min=1970,max=2100"`
	Description *string `json:"description,omitempty"`
}

// UpdateCurriculumStatusRequest represents payload to transition curriculum lifecycle.
type UpdateCurriculumStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=DRAFT ACTIVE INACTIVE ARCHIVED"`
}

// AddCurriculumCourseRequest represents payload to associate a course with a curriculum.
type AddCurriculumCourseRequest struct {
	CourseID uuid.UUID `json:"course_id" validate:"required"`
	Semester int       `json:"semester" validate:"required,min=1,max=14"`
	Category string    `json:"category" validate:"required,oneof=REQUIRED ELECTIVE"`
}

// UpdateCurriculumCourseRequest represents payload to update semester or category of a curriculum course.
type UpdateCurriculumCourseRequest struct {
	Semester *int    `json:"semester,omitempty" validate:"omitempty,min=1,max=14"`
	Category *string `json:"category,omitempty" validate:"omitempty,oneof=REQUIRED ELECTIVE"`
}

// CurriculumResponse represents standard API output for a curriculum entity.
type CurriculumResponse struct {
	ID             uuid.UUID        `json:"id"`
	StudyProgramID uuid.UUID        `json:"study_program_id"`
	Code           string           `json:"code"`
	Name           string           `json:"name"`
	Year           int              `json:"year"`
	Description    *string          `json:"description,omitempty"`
	Status         string           `json:"status"`
	Stats          *CurriculumStats `json:"stats,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// ListCurriculaFilter represents query filtering options for curricula.
type ListCurriculaFilter struct {
	Search         string
	StudyProgramID *uuid.UUID
	Status         string
	Year           *int
}

// ListCurriculumCoursesFilter represents query filter for courses in a curriculum.
type ListCurriculumCoursesFilter struct {
	Semester *int
	Category string
}
