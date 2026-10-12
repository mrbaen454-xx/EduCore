package study_programs

import (
	"time"

	"github.com/google/uuid"
)

// CreateStudyProgramRequest defines the payload for creating a study program.
type CreateStudyProgramRequest struct {
	FacultyID   uuid.UUID `json:"faculty_id" binding:"required"`
	Code        string    `json:"code" binding:"required,min=1,max=20"`
	Name        string    `json:"name" binding:"required,min=1,max=150"`
	Level       string    `json:"level" binding:"required,oneof=D3 D4 S1 S2 S3 Profesi"`
	Description *string   `json:"description,omitempty" binding:"omitempty,max=1000"`
}

// UpdateStudyProgramRequest defines the payload for updating a study program.
type UpdateStudyProgramRequest struct {
	FacultyID   *uuid.UUID `json:"faculty_id,omitempty"`
	Code        *string    `json:"code,omitempty" binding:"omitempty,min=1,max=20"`
	Name        *string    `json:"name,omitempty" binding:"omitempty,min=1,max=150"`
	Level       *string    `json:"level,omitempty" binding:"omitempty,oneof=D3 D4 S1 S2 S3 Profesi"`
	Description *string    `json:"description,omitempty" binding:"omitempty,max=1000"`
}

// UpdateStatusRequest defines the payload for activating/deactivating a study program.
type UpdateStatusRequest struct {
	IsActive *bool `json:"is_active" binding:"required"`
}

// FacultySummary represents a brief summary of the parent faculty.
type FacultySummary struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}

// StudyProgramResponse represents the API response format for a study program.
type StudyProgramResponse struct {
	ID          uuid.UUID      `json:"id"`
	Faculty     FacultySummary `json:"faculty"`
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	Level       string         `json:"level"`
	Description *string        `json:"description,omitempty"`
	IsActive    bool           `json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// StudyProgramFilter defines query filtering parameters for listing study programs.
type StudyProgramFilter struct {
	Search    string
	FacultyID *uuid.UUID
	Level     string
	IsActive  *bool
}
