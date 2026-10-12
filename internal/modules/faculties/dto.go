package faculties

import (
	"time"

	"github.com/google/uuid"
)

// CreateFacultyRequest defines the payload for creating a new faculty.
type CreateFacultyRequest struct {
	Code        string  `json:"code" binding:"required,min=1,max=20"`
	Name        string  `json:"name" binding:"required,min=1,max=150"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=1000"`
}

// UpdateFacultyRequest defines the payload for partially updating a faculty.
type UpdateFacultyRequest struct {
	Code        *string `json:"code,omitempty" binding:"omitempty,min=1,max=20"`
	Name        *string `json:"name,omitempty" binding:"omitempty,min=1,max=150"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=1000"`
}

// UpdateStatusRequest defines the payload for activating/deactivating a faculty.
type UpdateStatusRequest struct {
	IsActive *bool `json:"is_active" binding:"required"`
}

// FacultyResponse represents the API response body for faculty data.
type FacultyResponse struct {
	ID                uuid.UUID `json:"id"`
	Code              string    `json:"code"`
	Name              string    `json:"name"`
	Description       *string   `json:"description,omitempty"`
	IsActive          bool      `json:"is_active"`
	StudyProgramCount *int      `json:"study_program_count,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// FacultyFilter defines query filtering parameters for listing faculties.
type FacultyFilter struct {
	Search   string
	IsActive *bool
}
