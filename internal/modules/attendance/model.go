package attendance

import (
	"time"

	"github.com/google/uuid"
)

// ClassSession represents a lecture session / meeting of a class.
type ClassSession struct {
	ID            uuid.UUID  `json:"id"`
	ClassID       uuid.UUID  `json:"class_id"`
	SessionNumber int        `json:"session_number"`
	Date          time.Time  `json:"date"`
	StartTime     string     `json:"start_time"` // HH:MM or HH:MM:SS
	EndTime       string     `json:"end_time"`   // HH:MM or HH:MM:SS
	Topic         string     `json:"topic"`
	Remarks       *string    `json:"remarks,omitempty"`
	LecturerID    *uuid.UUID `json:"lecturer_id,omitempty"`
	Status        string     `json:"status"` // SCHEDULED, OPEN, COMPLETED, CANCELLED
	CreatedBy     *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// AttendanceSummary represents statistical counts of student presences for a session or class.
type AttendanceSummary struct {
	TotalEnrolled int `json:"total_enrolled"`
	TotalRecorded int `json:"total_recorded"`
	Present       int `json:"present"`
	Late          int `json:"late"`
	Excused       int `json:"excused"`
	Sick          int `json:"sick"`
	Absent        int `json:"absent"`
}

// ClassSessionDetail represents a lecture session enriched with class, course, and lecturer metadata.
type ClassSessionDetail struct {
	ClassSession
	ClassCode         string            `json:"class_code"`
	ClassName         string            `json:"class_name"`
	CourseCode        string            `json:"course_code"`
	CourseName        string            `json:"course_name"`
	CourseCredits     int               `json:"course_credits"`
	LecturerName      *string           `json:"lecturer_name,omitempty"`
	LecturerNIDN      *string           `json:"lecturer_nidn,omitempty"`
	AttendanceSummary AttendanceSummary `json:"attendance_summary"`
}

// Attendance represents an individual student's presence status for a session.
type Attendance struct {
	ID           uuid.UUID  `json:"id"`
	SessionID    uuid.UUID  `json:"session_id"`
	StudentID    uuid.UUID  `json:"student_id"`
	EnrollmentID uuid.UUID  `json:"enrollment_id"`
	Status       string     `json:"status"` // PRESENT, LATE, EXCUSED, SICK, ABSENT
	Remarks      *string    `json:"remarks,omitempty"`
	RecordedBy   *uuid.UUID `json:"recorded_by,omitempty"`
	RecordedAt   time.Time  `json:"recorded_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// AttendanceDetail represents an attendance record enriched with student and recorder names.
type AttendanceDetail struct {
	Attendance
	StudentNIM     string  `json:"student_nim"`
	StudentName    string  `json:"student_name"`
	RecordedByName *string `json:"recorded_by_name,omitempty"`
}

// SessionStudent represents an enrolled student in a session's roster with current presence info.
type SessionStudent struct {
	StudentID        uuid.UUID  `json:"student_id"`
	EnrollmentID     uuid.UUID  `json:"enrollment_id"`
	StudentNIM       string     `json:"student_nim"`
	StudentName      string     `json:"student_name"`
	EnrollmentStatus string     `json:"enrollment_status"`
	AttendanceID     *uuid.UUID `json:"attendance_id,omitempty"`
	AttendanceStatus *string    `json:"attendance_status,omitempty"`
	Remarks          *string    `json:"remarks,omitempty"`
	RecordedAt       *time.Time `json:"recorded_at,omitempty"`
}

// StudentAttendanceHistory represents a student's individual attendance log item.
type StudentAttendanceHistory struct {
	AttendanceID  uuid.UUID `json:"attendance_id"`
	SessionID     uuid.UUID `json:"session_id"`
	SessionNumber int       `json:"session_number"`
	Date          string    `json:"date"`
	StartTime     string    `json:"start_time"`
	EndTime       string    `json:"end_time"`
	Topic         string    `json:"topic"`
	ClassID       uuid.UUID `json:"class_id"`
	ClassCode     string    `json:"class_code"`
	ClassName     string    `json:"class_name"`
	CourseCode    string    `json:"course_code"`
	CourseName    string    `json:"course_name"`
	Status        string    `json:"status"`
	Remarks       *string   `json:"remarks,omitempty"`
	RecordedAt    string    `json:"recorded_at"`
}

// StudentClassSummary aggregates a student's presence counts within a specific class.
type StudentClassSummary struct {
	StudentID            uuid.UUID `json:"student_id"`
	StudentNIM           string    `json:"student_nim"`
	StudentName          string    `json:"student_name"`
	TotalSessions        int       `json:"total_sessions"`
	Present              int       `json:"present"`
	Late                 int       `json:"late"`
	Excused              int       `json:"excused"`
	Sick                 int       `json:"sick"`
	Absent               int       `json:"absent"`
	AttendancePercentage float64   `json:"attendance_percentage"`
}

// ClassAttendanceSummary represents the cumulative attendance report for a whole class.
type ClassAttendanceSummary struct {
	ClassID           uuid.UUID             `json:"class_id"`
	ClassCode         string                `json:"class_code"`
	ClassName         string                `json:"class_name"`
	TotalSessions     int                   `json:"total_sessions"`
	CompletedSessions int                   `json:"completed_sessions"`
	Students          []StudentClassSummary `json:"students"`
}
