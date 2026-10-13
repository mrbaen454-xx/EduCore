package academic_periods

import (
	"time"

	"github.com/google/uuid"
)

// AcademicPeriod represents an academic term/semester.
type AcademicPeriod struct {
	ID           uuid.UUID `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	SemesterType string    `json:"semester_type"` // ODD, EVEN, SHORT
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
