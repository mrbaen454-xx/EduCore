package academic_periods

import (
	"time"

	"github.com/google/uuid"
)

type CreateAcademicPeriodRequest struct {
	Code         string `json:"code" binding:"required,min=2,max=20"`
	Name         string `json:"name" binding:"required,min=2,max=100"`
	SemesterType string `json:"semester_type" binding:"required,oneof=ODD EVEN SHORT"`
	StartDate    string `json:"start_date" binding:"required"` // Format: YYYY-MM-DD
	EndDate      string `json:"end_date" binding:"required"`   // Format: YYYY-MM-DD
	IsActive     *bool  `json:"is_active"`
}

type UpdateAcademicPeriodRequest struct {
	Name         *string `json:"name" binding:"omitempty,min=2,max=100"`
	SemesterType *string `json:"semester_type" binding:"omitempty,oneof=ODD EVEN SHORT"`
	StartDate    *string `json:"start_date"`
	EndDate      *string `json:"end_date"`
	IsActive     *bool   `json:"is_active"`
}

type AcademicPeriodFilter struct {
	Search       string
	SemesterType string
	IsActive     *bool
}

type AcademicPeriodResponse struct {
	ID           uuid.UUID `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	SemesterType string    `json:"semester_type"`
	StartDate    string    `json:"start_date"`
	EndDate      string    `json:"end_date"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func ToResponse(p *AcademicPeriod) AcademicPeriodResponse {
	return AcademicPeriodResponse{
		ID:           p.ID,
		Code:         p.Code,
		Name:         p.Name,
		SemesterType: p.SemesterType,
		StartDate:    p.StartDate.Format("2006-01-02"),
		EndDate:      p.EndDate.Format("2006-01-02"),
		IsActive:     p.IsActive,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}
