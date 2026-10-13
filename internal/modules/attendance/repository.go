package attendance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"educore/internal/modules/lecturers"
	"educore/internal/modules/students"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSessionNotFound           = appErrors.ErrNotFound("Sesi perkuliahan tidak ditemukan")
	ErrAttendanceNotFound        = appErrors.ErrNotFound("Catatan presensi tidak ditemukan")
	ErrClassNotFound             = appErrors.ErrNotFound("Kelas perkuliahan tidak ditemukan")
	ErrStudentNotFound           = appErrors.ErrNotFound("Data mahasiswa tidak ditemukan")
	ErrLecturerNotFound          = appErrors.ErrNotFound("Data dosen tidak ditemukan")
	ErrDuplicateSessionNumber    = appErrors.ErrConflict("Nomor pertemuan sudah ada pada kelas ini")
	ErrDuplicateAttendance       = appErrors.ErrConflict("Presensi mahasiswa untuk sesi ini sudah dicatat")
	ErrSessionCancelled          = appErrors.ErrConflict("Sesi perkuliahan telah dibatalkan")
	ErrSessionClosed             = appErrors.ErrConflict("Sesi perkuliahan telah ditutup / selesai")
	ErrStudentNotEnrolledInClass = appErrors.ErrConflict("Mahasiswa tidak terdaftar sebagai peserta aktif pada kelas ini")
	ErrUnauthorizedLecturer      = appErrors.ErrForbidden("Anda bukan dosen yang ditugaskan pada kelas ini")
	ErrUnauthorizedAttendance    = appErrors.ErrForbidden("Anda tidak berhak mengakses presensi ini")
)

type Repository interface {
	// Verifications
	CheckLecturerTeachingClass(ctx context.Context, classID uuid.UUID, userID uuid.UUID) (bool, error)
	GetLecturerByUserID(ctx context.Context, userID uuid.UUID) (*lecturers.Lecturer, error)
	GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*students.Student, error)
	CheckStudentEnrolledInClass(ctx context.Context, classID, studentID uuid.UUID) (bool, error)

	// Session management
	CreateSession(ctx context.Context, s *ClassSession) error
	GetSessionByID(ctx context.Context, sessionID uuid.UUID) (*ClassSessionDetail, error)
	ListSessionsByClass(ctx context.Context, classID uuid.UUID, p pagination.Params) ([]ClassSessionDetail, int, error)
	UpdateSession(ctx context.Context, s *ClassSession) error
	CloseSession(ctx context.Context, sessionID uuid.UUID) error

	// Session roster & attendance records
	GetSessionStudents(ctx context.Context, sessionID uuid.UUID) ([]SessionStudent, error)
	GetSessionAttendances(ctx context.Context, sessionID uuid.UUID) ([]AttendanceDetail, AttendanceSummary, error)

	// Transactional attendance
	RecordBatchAttendanceTx(ctx context.Context, sessionID, recordedBy uuid.UUID, items []AttendanceItem) error
	UpdateAttendanceTx(ctx context.Context, sessionID, attendanceID, recordedBy uuid.UUID, status string, remarks *string) error

	// Student history
	GetStudentAttendanceHistory(ctx context.Context, studentID uuid.UUID, classID *uuid.UUID, p pagination.Params) ([]StudentAttendanceHistory, int, AttendanceSummary, error)

	// Class cumulative report
	GetClassAttendanceSummary(ctx context.Context, classID uuid.UUID) (*ClassAttendanceSummary, error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) CheckLecturerTeachingClass(ctx context.Context, classID, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM class_lecturers cl
			JOIN lecturers l ON l.id = cl.lecturer_id
			WHERE cl.class_id = $1 AND l.user_id = $2 AND cl.status = 'ACTIVE' AND l.status = 'ACTIVE'
		)
	`
	var isTeaching bool
	err := r.pool.QueryRow(ctx, query, classID, userID).Scan(&isTeaching)
	if err != nil {
		return false, fmt.Errorf("failed to check lecturer assignment: %w", err)
	}
	return isTeaching, nil
}

func (r *pgRepository) GetLecturerByUserID(ctx context.Context, userID uuid.UUID) (*lecturers.Lecturer, error) {
	query := `
		SELECT l.id, l.user_id, l.study_program_id, l.nidn, l.nip, l.name, l.title_prefix, l.title_suffix,
		       l.email, l.phone, l.status, l.created_at, l.updated_at
		FROM lecturers l
		WHERE l.user_id = $1
	`
	var l lecturers.Lecturer
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&l.ID, &l.UserID, &l.StudyProgramID, &l.NIDN, &l.NIP, &l.Name, &l.TitlePrefix, &l.TitleSuffix,
		&l.Email, &l.Phone, &l.Status, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLecturerNotFound
		}
		return nil, fmt.Errorf("failed to query lecturer by user id: %w", err)
	}
	return &l, nil
}

func (r *pgRepository) GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*students.Student, error) {
	query := `
		SELECT s.id, s.user_id, s.study_program_id, s.nim, s.name, s.email, s.phone, s.gender,
		       s.birth_date, s.admission_year, s.status, s.created_at, s.updated_at
		FROM students s
		WHERE s.user_id = $1
	`
	var s students.Student
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&s.ID, &s.UserID, &s.StudyProgramID, &s.NIM, &s.Name, &s.Email, &s.Phone, &s.Gender,
		&s.BirthDate, &s.AdmissionYear, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStudentNotFound
		}
		return nil, fmt.Errorf("failed to query student by user id: %w", err)
	}
	return &s, nil
}

func (r *pgRepository) CheckStudentEnrolledInClass(ctx context.Context, classID, studentID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM enrollments
			WHERE class_id = $1 AND student_id = $2 AND status = 'ENROLLED'
		)
	`
	var isEnrolled bool
	err := r.pool.QueryRow(ctx, query, classID, studentID).Scan(&isEnrolled)
	if err != nil {
		return false, fmt.Errorf("failed to check student enrollment: %w", err)
	}
	return isEnrolled, nil
}

func (r *pgRepository) CreateSession(ctx context.Context, s *ClassSession) error {
	query := `
		INSERT INTO class_sessions (
			class_id, session_number, date, start_time, end_time, topic, remarks, lecturer_id, status, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4::time, $5::time, $6, $7, $8, $9, $10, $11, $12
		)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		s.ClassID, s.SessionNumber, s.Date, s.StartTime, s.EndTime, s.Topic, s.Remarks, s.LecturerID, s.Status, s.CreatedBy, s.CreatedAt, s.UpdatedAt,
	).Scan(&s.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return ErrDuplicateSessionNumber
			}
			if pgErr.Code == pgerrcode.ForeignKeyViolation {
				if strings.Contains(pgErr.ConstraintName, "class") {
					return ErrClassNotFound
				}
				if strings.Contains(pgErr.ConstraintName, "lecturer") {
					return ErrLecturerNotFound
				}
			}
		}
		return fmt.Errorf("failed to insert class session: %w", err)
	}
	return nil
}

func (r *pgRepository) GetSessionByID(ctx context.Context, sessionID uuid.UUID) (*ClassSessionDetail, error) {
	query := `
		SELECT cs.id, cs.class_id, cs.session_number, cs.date, cs.start_time::text, cs.end_time::text,
		       cs.topic, cs.remarks, cs.lecturer_id, cs.status, cs.created_by, cs.created_at, cs.updated_at,
		       c.code, c.name,
		       cr.code, cr.name, cr.credits,
		       l.name, l.nidn
		FROM class_sessions cs
		JOIN classes c ON c.id = cs.class_id
		JOIN course_offerings co ON co.id = c.course_offering_id
		JOIN courses cr ON cr.id = co.course_id
		LEFT JOIN lecturers l ON l.id = cs.lecturer_id
		WHERE cs.id = $1
	`
	var d ClassSessionDetail
	err := r.pool.QueryRow(ctx, query, sessionID).Scan(
		&d.ID, &d.ClassID, &d.SessionNumber, &d.Date, &d.StartTime, &d.EndTime,
		&d.Topic, &d.Remarks, &d.LecturerID, &d.Status, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt,
		&d.ClassCode, &d.ClassName,
		&d.CourseCode, &d.CourseName, &d.CourseCredits,
		&d.LecturerName, &d.LecturerNIDN,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to query session by id: %w", err)
	}

	summary, err := r.getSessionSummary(ctx, d.ID, d.ClassID)
	if err != nil {
		return nil, err
	}
	d.AttendanceSummary = summary

	return &d, nil
}

func (r *pgRepository) ListSessionsByClass(ctx context.Context, classID uuid.UUID, p pagination.Params) ([]ClassSessionDetail, int, error) {
	countQuery := `SELECT COUNT(*) FROM class_sessions WHERE class_id = $1`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, classID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	selectQuery := `
		SELECT cs.id, cs.class_id, cs.session_number, cs.date, cs.start_time::text, cs.end_time::text,
		       cs.topic, cs.remarks, cs.lecturer_id, cs.status, cs.created_by, cs.created_at, cs.updated_at,
		       c.code, c.name,
		       cr.code, cr.name, cr.credits,
		       l.name, l.nidn
		FROM class_sessions cs
		JOIN classes c ON c.id = cs.class_id
		JOIN course_offerings co ON co.id = c.course_offering_id
		JOIN courses cr ON cr.id = co.course_id
		LEFT JOIN lecturers l ON l.id = cs.lecturer_id
		WHERE cs.class_id = $1
		ORDER BY cs.session_number ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, selectQuery, classID, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []ClassSessionDetail
	for rows.Next() {
		var d ClassSessionDetail
		if err := rows.Scan(
			&d.ID, &d.ClassID, &d.SessionNumber, &d.Date, &d.StartTime, &d.EndTime,
			&d.Topic, &d.Remarks, &d.LecturerID, &d.Status, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt,
			&d.ClassCode, &d.ClassName,
			&d.CourseCode, &d.CourseName, &d.CourseCredits,
			&d.LecturerName, &d.LecturerNIDN,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan session row: %w", err)
		}
		sessions = append(sessions, d)
	}

	for i := range sessions {
		summary, err := r.getSessionSummary(ctx, sessions[i].ID, sessions[i].ClassID)
		if err != nil {
			return nil, 0, err
		}
		sessions[i].AttendanceSummary = summary
	}

	return sessions, total, nil
}

func (r *pgRepository) UpdateSession(ctx context.Context, s *ClassSession) error {
	query := `
		UPDATE class_sessions
		SET session_number = $1, date = $2, start_time = $3::time, end_time = $4::time,
		    topic = $5, remarks = $6, lecturer_id = $7, status = $8, updated_at = $9
		WHERE id = $10
	`
	res, err := r.pool.Exec(ctx, query,
		s.SessionNumber, s.Date, s.StartTime, s.EndTime,
		s.Topic, s.Remarks, s.LecturerID, s.Status, time.Now(), s.ID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return ErrDuplicateSessionNumber
		}
		return fmt.Errorf("failed to update session: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (r *pgRepository) CloseSession(ctx context.Context, sessionID uuid.UUID) error {
	query := `
		UPDATE class_sessions
		SET status = 'COMPLETED', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status != 'CANCELLED'
	`
	res, err := r.pool.Exec(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (r *pgRepository) GetSessionStudents(ctx context.Context, sessionID uuid.UUID) ([]SessionStudent, error) {
	// First get class_id of session
	var classID uuid.UUID
	err := r.pool.QueryRow(ctx, "SELECT class_id FROM class_sessions WHERE id = $1", sessionID).Scan(&classID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get class id for session: %w", err)
	}

	// Query active enrolled students with left join on attendances for this session
	query := `
		SELECT s.id, e.id, s.nim, s.name, e.status,
		       a.id, a.status, a.remarks, a.recorded_at
		FROM enrollments e
		JOIN students s ON s.id = e.student_id
		LEFT JOIN attendances a ON a.session_id = $1 AND a.student_id = s.id
		WHERE e.class_id = $2 AND e.status = 'ENROLLED'
		ORDER BY s.nim ASC
	`
	rows, err := r.pool.Query(ctx, query, sessionID, classID)
	if err != nil {
		return nil, fmt.Errorf("failed to query session students: %w", err)
	}
	defer rows.Close()

	var list []SessionStudent
	for rows.Next() {
		var ss SessionStudent
		if err := rows.Scan(
			&ss.StudentID, &ss.EnrollmentID, &ss.StudentNIM, &ss.StudentName, &ss.EnrollmentStatus,
			&ss.AttendanceID, &ss.AttendanceStatus, &ss.Remarks, &ss.RecordedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan session student: %w", err)
		}
		list = append(list, ss)
	}

	return list, nil
}

func (r *pgRepository) GetSessionAttendances(ctx context.Context, sessionID uuid.UUID) ([]AttendanceDetail, AttendanceSummary, error) {
	var classID uuid.UUID
	err := r.pool.QueryRow(ctx, "SELECT class_id FROM class_sessions WHERE id = $1", sessionID).Scan(&classID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, AttendanceSummary{}, ErrSessionNotFound
		}
		return nil, AttendanceSummary{}, fmt.Errorf("failed to get session: %w", err)
	}

	query := `
		SELECT a.id, a.session_id, a.student_id, a.enrollment_id, a.status, a.remarks,
		       a.recorded_by, a.recorded_at, a.created_at, a.updated_at,
		       s.nim, s.name,
		       u.name
		FROM attendances a
		JOIN students s ON s.id = a.student_id
		LEFT JOIN users u ON u.id = a.recorded_by
		WHERE a.session_id = $1
		ORDER BY s.nim ASC
	`
	rows, err := r.pool.Query(ctx, query, sessionID)
	if err != nil {
		return nil, AttendanceSummary{}, fmt.Errorf("failed to query attendances: %w", err)
	}
	defer rows.Close()

	var list []AttendanceDetail
	for rows.Next() {
		var d AttendanceDetail
		if err := rows.Scan(
			&d.ID, &d.SessionID, &d.StudentID, &d.EnrollmentID, &d.Status, &d.Remarks,
			&d.RecordedBy, &d.RecordedAt, &d.CreatedAt, &d.UpdatedAt,
			&d.StudentNIM, &d.StudentName,
			&d.RecordedByName,
		); err != nil {
			return nil, AttendanceSummary{}, fmt.Errorf("failed to scan attendance: %w", err)
		}
		list = append(list, d)
	}

	summary, err := r.getSessionSummary(ctx, sessionID, classID)
	if err != nil {
		return nil, AttendanceSummary{}, err
	}

	return list, summary, nil
}

func (r *pgRepository) RecordBatchAttendanceTx(ctx context.Context, sessionID, recordedBy uuid.UUID, items []AttendanceItem) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Lock and check session
	var classID uuid.UUID
	var sessionStatus string
	sessionQuery := `
		SELECT class_id, status
		FROM class_sessions
		WHERE id = $1
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, sessionQuery, sessionID).Scan(&classID, &sessionStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("failed to lock session: %w", err)
	}

	if sessionStatus == "CANCELLED" {
		return ErrSessionCancelled
	}

	// If session was SCHEDULED, transition to OPEN
	if sessionStatus == "SCHEDULED" {
		_, _ = tx.Exec(ctx, "UPDATE class_sessions SET status = 'OPEN', updated_at = CURRENT_TIMESTAMP WHERE id = $1", sessionID)
	}

	// 2. Fetch active enrollments for this class
	enrollQuery := `SELECT student_id, id FROM enrollments WHERE class_id = $1 AND status = 'ENROLLED'`
	eRows, err := tx.Query(ctx, enrollQuery, classID)
	if err != nil {
		return fmt.Errorf("failed to query active enrollments: %w", err)
	}
	defer eRows.Close()

	studentEnrollments := make(map[uuid.UUID]uuid.UUID)
	for eRows.Next() {
		var sID, eID uuid.UUID
		if err := eRows.Scan(&sID, &eID); err != nil {
			return fmt.Errorf("failed to scan enrollment: %w", err)
		}
		studentEnrollments[sID] = eID
	}
	eRows.Close()

	// 3. Upsert attendance records atomically
	upsertQuery := `
		INSERT INTO attendances (session_id, student_id, enrollment_id, status, remarks, recorded_by, recorded_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (session_id, student_id) DO UPDATE
		SET status = EXCLUDED.status,
		    remarks = EXCLUDED.remarks,
		    recorded_by = EXCLUDED.recorded_by,
		    recorded_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
	`

	for _, item := range items {
		enrollmentID, ok := studentEnrollments[item.StudentID]
		if !ok {
			return appErrors.ErrConflict(fmt.Sprintf("Mahasiswa dengan ID '%s' tidak terdaftar sebagai peserta aktif di kelas ini", item.StudentID))
		}

		if _, err := tx.Exec(ctx, upsertQuery, sessionID, item.StudentID, enrollmentID, item.Status, item.Remarks, recordedBy); err != nil {
			return fmt.Errorf("failed to upsert attendance for student %s: %w", item.StudentID, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *pgRepository) UpdateAttendanceTx(ctx context.Context, sessionID, attendanceID, recordedBy uuid.UUID, status string, remarks *string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Lock and check session
	var sessionStatus string
	err = tx.QueryRow(ctx, "SELECT status FROM class_sessions WHERE id = $1 FOR UPDATE", sessionID).Scan(&sessionStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("failed to lock session: %w", err)
	}

	if sessionStatus == "CANCELLED" {
		return ErrSessionCancelled
	}

	// 2. Lock and update attendance
	updateQuery := `
		UPDATE attendances
		SET status = $1, remarks = $2, recorded_by = $3, recorded_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4 AND session_id = $5
	`
	res, err := tx.Exec(ctx, updateQuery, status, remarks, recordedBy, attendanceID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update attendance: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrAttendanceNotFound
	}

	return tx.Commit(ctx)
}

func (r *pgRepository) GetStudentAttendanceHistory(ctx context.Context, studentID uuid.UUID, classID *uuid.UUID, p pagination.Params) ([]StudentAttendanceHistory, int, AttendanceSummary, error) {
	whereClauses := []string{"a.student_id = $1"}
	args := []any{studentID}
	argIdx := 2

	if classID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("cs.class_id = $%d", argIdx))
		args = append(args, *classID)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM attendances a
		JOIN class_sessions cs ON cs.id = a.session_id
		WHERE %s
	`, whereSQL)

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, AttendanceSummary{}, fmt.Errorf("failed to count student attendance: %w", err)
	}

	selectQuery := fmt.Sprintf(`
		SELECT a.id, a.session_id, cs.session_number, cs.date::text, cs.start_time::text, cs.end_time::text,
		       cs.topic, cs.class_id, c.code, c.name, cr.code, cr.name,
		       a.status, a.remarks, a.recorded_at::text
		FROM attendances a
		JOIN class_sessions cs ON cs.id = a.session_id
		JOIN classes c ON c.id = cs.class_id
		JOIN course_offerings co ON co.id = c.course_offering_id
		JOIN courses cr ON cr.id = co.course_id
		WHERE %s
		ORDER BY cs.date DESC, cs.session_number DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	argsWithPaging := append(args, p.Limit, p.Offset())
	rows, err := r.pool.Query(ctx, selectQuery, argsWithPaging...)
	if err != nil {
		return nil, 0, AttendanceSummary{}, fmt.Errorf("failed to query student attendance history: %w", err)
	}
	defer rows.Close()

	var history []StudentAttendanceHistory
	for rows.Next() {
		var h StudentAttendanceHistory
		if err := rows.Scan(
			&h.AttendanceID, &h.SessionID, &h.SessionNumber, &h.Date, &h.StartTime, &h.EndTime,
			&h.Topic, &h.ClassID, &h.ClassCode, &h.ClassName, &h.CourseCode, &h.CourseName,
			&h.Status, &h.Remarks, &h.RecordedAt,
		); err != nil {
			return nil, 0, AttendanceSummary{}, fmt.Errorf("failed to scan student attendance item: %w", err)
		}
		history = append(history, h)
	}

	// Aggregate summary counts for this student
	summaryQuery := fmt.Sprintf(`
		SELECT COUNT(*),
		       COALESCE(SUM(CASE WHEN a.status = 'PRESENT' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN a.status = 'LATE' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN a.status = 'EXCUSED' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN a.status = 'SICK' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN a.status = 'ABSENT' THEN 1 ELSE 0 END), 0)
		FROM attendances a
		JOIN class_sessions cs ON cs.id = a.session_id
		WHERE %s
	`, whereSQL)

	var summary AttendanceSummary
	_ = r.pool.QueryRow(ctx, summaryQuery, args...).Scan(
		&summary.TotalRecorded,
		&summary.Present,
		&summary.Late,
		&summary.Excused,
		&summary.Sick,
		&summary.Absent,
	)

	return history, total, summary, nil
}

func (r *pgRepository) GetClassAttendanceSummary(ctx context.Context, classID uuid.UUID) (*ClassAttendanceSummary, error) {
	var cas ClassAttendanceSummary
	cas.ClassID = classID

	// 1. Fetch class info
	classQuery := `SELECT code, name FROM classes WHERE id = $1`
	err := r.pool.QueryRow(ctx, classQuery, classID).Scan(&cas.ClassCode, &cas.ClassName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrClassNotFound
		}
		return nil, fmt.Errorf("failed to query class: %w", err)
	}

	// 2. Count total & completed sessions
	sessCountQuery := `
		SELECT COUNT(*),
		       COALESCE(SUM(CASE WHEN status = 'COMPLETED' THEN 1 ELSE 0 END), 0)
		FROM class_sessions
		WHERE class_id = $1 AND status != 'CANCELLED'
	`
	_ = r.pool.QueryRow(ctx, sessCountQuery, classID).Scan(&cas.TotalSessions, &cas.CompletedSessions)

	// 3. Aggregate each active student's attendance stats
	studentsQuery := `
		SELECT s.id, s.nim, s.name,
		       COALESCE(SUM(CASE WHEN a.status = 'PRESENT' THEN 1 ELSE 0 END), 0) as present_count,
		       COALESCE(SUM(CASE WHEN a.status = 'LATE' THEN 1 ELSE 0 END), 0) as late_count,
		       COALESCE(SUM(CASE WHEN a.status = 'EXCUSED' THEN 1 ELSE 0 END), 0) as excused_count,
		       COALESCE(SUM(CASE WHEN a.status = 'SICK' THEN 1 ELSE 0 END), 0) as sick_count,
		       COALESCE(SUM(CASE WHEN a.status = 'ABSENT' THEN 1 ELSE 0 END), 0) as absent_count
		FROM enrollments e
		JOIN students s ON s.id = e.student_id
		LEFT JOIN class_sessions cs ON cs.class_id = e.class_id AND cs.status != 'CANCELLED'
		LEFT JOIN attendances a ON a.session_id = cs.id AND a.student_id = s.id
		WHERE e.class_id = $1 AND e.status = 'ENROLLED'
		GROUP BY s.id, s.nim, s.name
		ORDER BY s.nim ASC
	`
	rows, err := r.pool.Query(ctx, studentsQuery, classID)
	if err != nil {
		return nil, fmt.Errorf("failed to query class student summaries: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var scs StudentClassSummary
		scs.TotalSessions = cas.TotalSessions
		if err := rows.Scan(
			&scs.StudentID, &scs.StudentNIM, &scs.StudentName,
			&scs.Present, &scs.Late, &scs.Excused, &scs.Sick, &scs.Absent,
		); err != nil {
			return nil, fmt.Errorf("failed to scan student class summary: %w", err)
		}
		if cas.TotalSessions > 0 {
			scs.AttendancePercentage = float64(scs.Present+scs.Late) / float64(cas.TotalSessions) * 100.0
		}
		cas.Students = append(cas.Students, scs)
	}

	return &cas, nil
}

func (r *pgRepository) getSessionSummary(ctx context.Context, sessionID, classID uuid.UUID) (AttendanceSummary, error) {
	var summary AttendanceSummary

	// Total enrolled in class
	enrolledQuery := `SELECT COUNT(*) FROM enrollments WHERE class_id = $1 AND status = 'ENROLLED'`
	_ = r.pool.QueryRow(ctx, enrolledQuery, classID).Scan(&summary.TotalEnrolled)

	// Attendance breakdown for this session
	breakdownQuery := `
		SELECT COUNT(*),
		       COALESCE(SUM(CASE WHEN status = 'PRESENT' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN status = 'LATE' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN status = 'EXCUSED' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN status = 'SICK' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN status = 'ABSENT' THEN 1 ELSE 0 END), 0)
		FROM attendances
		WHERE session_id = $1
	`
	err := r.pool.QueryRow(ctx, breakdownQuery, sessionID).Scan(
		&summary.TotalRecorded,
		&summary.Present,
		&summary.Late,
		&summary.Excused,
		&summary.Sick,
		&summary.Absent,
	)
	if err != nil {
		return summary, fmt.Errorf("failed to query session summary breakdown: %w", err)
	}

	return summary, nil
}
