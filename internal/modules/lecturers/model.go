package lecturers

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Lecturer represents the domain model for an academic faculty member.
type Lecturer struct {
	ID             uuid.UUID  `json:"id"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	StudyProgramID uuid.UUID  `json:"study_program_id"`
	NIDN           string     `json:"nidn"`
	NIP            *string    `json:"nip,omitempty"`
	Name           string     `json:"name"`
	TitlePrefix    *string    `json:"title_prefix,omitempty"`
	TitleSuffix    *string    `json:"title_suffix,omitempty"`
	Email          string     `json:"email"`
	Phone          *string    `json:"phone,omitempty"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Relational fields populated from joins
	StudyProgramCode string `json:"study_program_code,omitempty"`
	StudyProgramName string `json:"study_program_name,omitempty"`
	FacultyName      string `json:"faculty_name,omitempty"`
}

// FullNameWithTitles formats the lecturer's name with prefix and suffix academic titles.
func (l *Lecturer) FullNameWithTitles() string {
	parts := []string{}
	if l.TitlePrefix != nil && strings.TrimSpace(*l.TitlePrefix) != "" {
		parts = append(parts, strings.TrimSpace(*l.TitlePrefix))
	}
	parts = append(parts, l.Name)
	res := strings.Join(parts, " ")
	if l.TitleSuffix != nil && strings.TrimSpace(*l.TitleSuffix) != "" {
		res = fmt.Sprintf("%s, %s", res, strings.TrimSpace(*l.TitleSuffix))
	}
	return res
}
