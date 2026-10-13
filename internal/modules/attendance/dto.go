package attendance

import (
	"github.com/google/uuid"
)

// CreateSessionRequest represents input for creating a new class session.
type CreateSessionRequest struct {
	SessionNumber int        `json:"session_number" binding:"required,min=1"`
	Date          string     `json:"date" binding:"required"`       // Format: YYYY-MM-DD
	StartTime     string     `json:"start_time" binding:"required"` // Format: HH:MM or HH:MM:SS
	EndTime       string     `json:"end_time" binding:"required"`   // Format: HH:MM or HH:MM:SS
	Topic         string     `json:"topic" binding:"required"`
	Remarks       *string    `json:"remarks,omitempty"`
	LecturerID    *uuid.UUID `json:"lecturer_id,omitempty"`
	Status        *string    `json:"status,omitempty"` // SCHEDULED or OPEN (default: OPEN)
}

// UpdateSessionRequest represents input for updating session metadata.
type UpdateSessionRequest struct {
	SessionNumber *int       `json:"session_number,omitempty" binding:"omitempty,min=1"`
	Date          *string    `json:"date,omitempty"`
	StartTime     *string    `json:"start_time,omitempty"`
	EndTime       *string    `json:"end_time,omitempty"`
	Topic         *string    `json:"topic,omitempty"`
	Remarks       *string    `json:"remarks,omitempty"`
	LecturerID    *uuid.UUID `json:"lecturer_id,omitempty"`
	Status        *string    `json:"status,omitempty"`
}

// AttendanceItem represents an individual entry in a batch attendance submission.
type AttendanceItem struct {
	StudentID uuid.UUID `json:"student_id" binding:"required"`
	Status    string    `json:"status" binding:"required,oneof=PRESENT LATE EXCUSED SICK ABSENT"`
	Remarks   *string   `json:"remarks,omitempty"`
}

// RecordBatchAttendanceRequest represents the payload for bulk recording student attendance.
type RecordBatchAttendanceRequest struct {
	Items []AttendanceItem `json:"items" binding:"required,min=1"`
}

// UpdateAttendanceRequest represents payload for correcting an individual student attendance record.
type UpdateAttendanceRequest struct {
	Status  string  `json:"status" binding:"required,oneof=PRESENT LATE EXCUSED SICK ABSENT"`
	Remarks *string `json:"remarks,omitempty"`
}

// SessionResponse represents the API response for session metadata.
type SessionResponse struct {
	ID                uuid.UUID         `json:"id"`
	ClassID           uuid.UUID         `json:"class_id"`
	ClassCode         string            `json:"class_code,omitempty"`
	ClassName         string            `json:"class_name,omitempty"`
	CourseCode        string            `json:"course_code,omitempty"`
	CourseName        string            `json:"course_name,omitempty"`
	SessionNumber     int               `json:"session_number"`
	Date              string            `json:"date"`
	StartTime         string            `json:"start_time"`
	EndTime           string            `json:"end_time"`
	Topic             string            `json:"topic"`
	Remarks           *string           `json:"remarks,omitempty"`
	LecturerID        *uuid.UUID        `json:"lecturer_id,omitempty"`
	LecturerName      *string           `json:"lecturer_name,omitempty"`
	LecturerNIDN      *string           `json:"lecturer_nidn,omitempty"`
	Status            string            `json:"status"`
	AttendanceSummary AttendanceSummary `json:"attendance_summary"`
	CreatedAt         string            `json:"created_at"`
	UpdatedAt         string            `json:"updated_at"`
}

// AttendanceDetailResponse represents API response for an individual attendance record.
type AttendanceDetailResponse struct {
	ID             uuid.UUID `json:"id"`
	SessionID      uuid.UUID `json:"session_id"`
	StudentID      uuid.UUID `json:"student_id"`
	StudentNIM     string    `json:"student_nim"`
	StudentName    string    `json:"student_name"`
	EnrollmentID   uuid.UUID `json:"enrollment_id"`
	Status         string    `json:"status"`
	Remarks        *string   `json:"remarks,omitempty"`
	RecordedByName *string   `json:"recorded_by_name,omitempty"`
	RecordedAt     string    `json:"recorded_at"`
	CreatedAt      string    `json:"created_at"`
	UpdatedAt      string    `json:"updated_at"`
}

// SessionRosterResponse represents the student roster of a session with current presence.
type SessionRosterResponse struct {
	SessionID        uuid.UUID        `json:"session_id"`
	SessionNumber    int              `json:"session_number"`
	ClassID          uuid.UUID        `json:"class_id"`
	TotalStudents    int              `json:"total_students"`
	RecordedStudents int              `json:"recorded_students"`
	Students         []SessionStudent `json:"students"`
}
