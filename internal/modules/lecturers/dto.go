package lecturers

import (
	"time"

	"github.com/google/uuid"
)

// CreateLecturerRequest defines the payload for creating a new lecturer.
type CreateLecturerRequest struct {
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	StudyProgramID uuid.UUID  `json:"study_program_id" binding:"required"`
	NIDN           string     `json:"nidn" binding:"required,min=5,max=30"`
	NIP            *string    `json:"nip,omitempty" binding:"omitempty,max=30"`
	Name           string     `json:"name" binding:"required,min=1,max=150"`
	TitlePrefix    *string    `json:"title_prefix,omitempty" binding:"omitempty,max=50"`
	TitleSuffix    *string    `json:"title_suffix,omitempty" binding:"omitempty,max=50"`
	Email          string     `json:"email" binding:"required,email,max=255"`
	Phone          *string    `json:"phone,omitempty" binding:"omitempty,max=25"`
}

// UpdateLecturerRequest defines the payload for partially updating a lecturer.
type UpdateLecturerRequest struct {
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	StudyProgramID *uuid.UUID `json:"study_program_id,omitempty"`
	NIDN           *string    `json:"nidn,omitempty" binding:"omitempty,min=5,max=30"`
	NIP            *string    `json:"nip,omitempty" binding:"omitempty,max=30"`
	Name           *string    `json:"name,omitempty" binding:"omitempty,min=1,max=150"`
	TitlePrefix    *string    `json:"title_prefix,omitempty" binding:"omitempty,max=50"`
	TitleSuffix    *string    `json:"title_suffix,omitempty" binding:"omitempty,max=50"`
	Email          *string    `json:"email,omitempty" binding:"omitempty,email,max=255"`
	Phone          *string    `json:"phone,omitempty" binding:"omitempty,max=25"`
}

// UpdateStatusRequest defines the payload for changing a lecturer's status.
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=ACTIVE INACTIVE LEAVE"`
}

// StudyProgramSummary represents a brief summary of the lecturer's home department.
type StudyProgramSummary struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}

// LecturerResponse represents the API response format for a lecturer.
type LecturerResponse struct {
	ID           uuid.UUID           `json:"id"`
	UserID       *uuid.UUID          `json:"user_id,omitempty"`
	StudyProgram StudyProgramSummary `json:"study_program"`
	NIDN         string              `json:"nidn"`
	NIP          *string             `json:"nip,omitempty"`
	Name         string              `json:"name"`
	FullName     string              `json:"full_name"`
	TitlePrefix  *string             `json:"title_prefix,omitempty"`
	TitleSuffix  *string             `json:"title_suffix,omitempty"`
	Email        string              `json:"email"`
	Phone        *string             `json:"phone,omitempty"`
	Status       string              `json:"status"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

// LecturerFilter defines query parameters for filtering lecturers.
type LecturerFilter struct {
	Search         string
	StudyProgramID *uuid.UUID
	Status         string
}
