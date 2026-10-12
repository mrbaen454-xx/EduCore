package students

import (
	"time"

	"github.com/google/uuid"
)

// CreateStudentRequest defines the payload for creating a new student profile.
type CreateStudentRequest struct {
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	StudyProgramID uuid.UUID  `json:"study_program_id" binding:"required"`
	NIM            string     `json:"nim" binding:"required,min=5,max=30"`
	Name           string     `json:"name" binding:"required,min=1,max=150"`
	Email          string     `json:"email" binding:"required,email,max=255"`
	Phone          *string    `json:"phone,omitempty" binding:"omitempty,max=25"`
	Gender         *string    `json:"gender,omitempty" binding:"omitempty,oneof=MALE FEMALE"`
	BirthDate      *string    `json:"birth_date,omitempty" binding:"omitempty"` // Format: YYYY-MM-DD
	AdmissionYear  int        `json:"admission_year" binding:"required,min=1970,max=2100"`
}

// UpdateStudentRequest defines the payload for updating an existing student.
type UpdateStudentRequest struct {
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	StudyProgramID *uuid.UUID `json:"study_program_id,omitempty"`
	NIM            *string    `json:"nim,omitempty" binding:"omitempty,min=5,max=30"`
	Name           *string    `json:"name,omitempty" binding:"omitempty,min=1,max=150"`
	Email          *string    `json:"email,omitempty" binding:"omitempty,email,max=255"`
	Phone          *string    `json:"phone,omitempty" binding:"omitempty,max=25"`
	Gender         *string    `json:"gender,omitempty" binding:"omitempty,oneof=MALE FEMALE"`
	BirthDate      *string    `json:"birth_date,omitempty" binding:"omitempty"`
	AdmissionYear  *int       `json:"admission_year,omitempty" binding:"omitempty,min=1970,max=2100"`
}

// UpdateStatusRequest defines the payload for changing a student's status.
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=ACTIVE INACTIVE PROBATION GRADUATED DROPOUT SUSPENDED"`
}

// StudyProgramSummary represents a brief summary of the student's study program.
type StudyProgramSummary struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}

// StudentResponse represents the API response format for a student.
type StudentResponse struct {
	ID            uuid.UUID           `json:"id"`
	UserID        *uuid.UUID          `json:"user_id,omitempty"`
	StudyProgram  StudyProgramSummary `json:"study_program"`
	NIM           string              `json:"nim"`
	Name          string              `json:"name"`
	Email         string              `json:"email"`
	Phone         *string             `json:"phone,omitempty"`
	Gender        *string             `json:"gender,omitempty"`
	BirthDate     *string             `json:"birth_date,omitempty"`
	AdmissionYear int                 `json:"admission_year"`
	Status        string              `json:"status"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

// StudentFilter defines query parameters for filtering students.
type StudentFilter struct {
	Search         string
	StudyProgramID *uuid.UUID
	AdmissionYear  *int
	Status         string
}
