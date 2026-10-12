package curricula

import (
	"time"

	"github.com/google/uuid"
)

// Valid curriculum status constants.
const (
	StatusDraft    = "DRAFT"
	StatusActive   = "ACTIVE"
	StatusInactive = "INACTIVE"
	StatusArchived = "ARCHIVED"
)

// Valid curriculum course categories.
const (
	CategoryRequired = "REQUIRED"
	CategoryElective = "ELECTIVE"
)

// Curriculum represents the domain model for a study program's curriculum.
type Curriculum struct {
	ID             uuid.UUID `json:"id"`
	StudyProgramID uuid.UUID `json:"study_program_id"`
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	Year           int       `json:"year"`
	Description    *string   `json:"description,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CurriculumCourse represents an association of a course within a curriculum.
type CurriculumCourse struct {
	ID           uuid.UUID `json:"id"`
	CurriculumID uuid.UUID `json:"curriculum_id"`
	CourseID     uuid.UUID `json:"course_id"`
	Semester     int       `json:"semester"`
	Category     string    `json:"category"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CurriculumCourseDetail represents a joined course view in a curriculum.
type CurriculumCourseDetail struct {
	ID           uuid.UUID `json:"id"`
	CurriculumID uuid.UUID `json:"curriculum_id"`
	CourseID     uuid.UUID `json:"course_id"`
	CourseCode   string    `json:"course_code"`
	CourseName   string    `json:"course_name"`
	Credits      int       `json:"credits"`
	Semester     int       `json:"semester"`
	Category     string    `json:"category"`
}

// CurriculumStats contains aggregated course credits metrics for a curriculum.
type CurriculumStats struct {
	TotalCourses    int `json:"total_courses"`
	TotalCredits    int `json:"total_credits"`
	RequiredCredits int `json:"required_credits"`
	ElectiveCredits int `json:"elective_credits"`
}
