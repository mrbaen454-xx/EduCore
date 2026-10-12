package study_programs

import (
	"time"

	"github.com/google/uuid"
)

// StudyProgram represents the domain model for an academic study program.
type StudyProgram struct {
	ID          uuid.UUID `json:"id"`
	FacultyID   uuid.UUID `json:"faculty_id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Level       string    `json:"level"`
	Description *string   `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Joined relational fields
	FacultyCode string `json:"faculty_code,omitempty"`
	FacultyName string `json:"faculty_name,omitempty"`
}
