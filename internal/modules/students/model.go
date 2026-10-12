package students

import (
	"time"

	"github.com/google/uuid"
)

// Student represents the domain model for an academic student.
type Student struct {
	ID             uuid.UUID  `json:"id"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	StudyProgramID uuid.UUID  `json:"study_program_id"`
	NIM            string     `json:"nim"`
	Name           string     `json:"name"`
	Email          string     `json:"email"`
	Phone          *string    `json:"phone,omitempty"`
	Gender         *string    `json:"gender,omitempty"`
	BirthDate      *time.Time `json:"birth_date,omitempty"`
	AdmissionYear  int        `json:"admission_year"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Relational fields populated from joins
	StudyProgramCode string `json:"study_program_code,omitempty"`
	StudyProgramName string `json:"study_program_name,omitempty"`
	FacultyName      string `json:"faculty_name,omitempty"`
}
