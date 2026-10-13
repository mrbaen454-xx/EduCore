package study_plans

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"educore/internal/modules/classes"
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
	ErrStudyPlanNotFound      = appErrors.ErrNotFound("KRS tidak ditemukan")
	ErrEnrollmentNotFound     = appErrors.ErrNotFound("Registrasi kelas tidak ditemukan")
	ErrStudentNotFound        = appErrors.ErrNotFound("Data mahasiswa tidak ditemukan")
	ErrAcademicPeriodNotFound = appErrors.ErrNotFound("Periode akademik tidak ditemukan")
	ErrClassNotFound          = appErrors.ErrNotFound("Kelas perkuliahan tidak ditemukan")

	ErrStudyPlanNotActive      = appErrors.ErrConflict("Status KRS tidak aktif untuk perubahan data")
	ErrAcademicPeriodNotActive = appErrors.ErrConflict("Periode akademik tidak aktif untuk pengisian KRS")
	ErrStudentNotActive        = appErrors.ErrConflict("Status mahasiswa tidak memenuhi syarat untuk pengisian KRS")
	ErrPeriodMismatch          = appErrors.ErrConflict("Kelas tidak berasal dari periode akademik yang sama dengan KRS")
	ErrClassNotOpen            = appErrors.ErrConflict("Kelas perkuliahan tidak dalam status OPEN")
	ErrOfferingNotOpen         = appErrors.ErrConflict("Penawaran mata kuliah tidak dalam status OPEN")
	ErrClassCapacityFull       = appErrors.ErrConflict("Kapasitas kelas telah penuh")
	ErrAlreadyEnrolledInClass  = appErrors.ErrConflict("Mahasiswa sudah terdaftar pada kelas ini")
	ErrAlreadyEnrolledInCourse = appErrors.ErrConflict("Mahasiswa sudah mengambil kelas lain untuk mata kuliah yang sama pada periode ini")
	ErrScheduleConflict        = appErrors.ErrConflict("Jadwal perkuliahan bentrok dengan jadwal kelas lain dalam KRS")
	ErrMaxCreditsExceeded      = appErrors.ErrConflict("Total SKS melebihi batas maksimum 24 SKS per semester")
	ErrEnrollmentNotActive     = appErrors.ErrConflict("Registrasi kelas tidak berstatus aktif untuk dibatalkan")
	ErrDuplicateStudyPlan      = appErrors.ErrConflict("KRS untuk mahasiswa dan periode akademik ini sudah ada")
	ErrUnauthorizedStudyPlan   = appErrors.ErrForbidden("Anda tidak berhak mengakses KRS ini")
)

type Repository interface {
	// Student profile lookup
	GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*students.Student, error)
	GetStudentByID(ctx context.Context, studentID uuid.UUID) (*students.Student, error)
	CheckAcademicPeriodActive(ctx context.Context, periodID uuid.UUID) (exists bool, active bool, err error)
	GetActiveAcademicPeriodID(ctx context.Context) (uuid.UUID, error)

	// Study plans
	CreateStudyPlan(ctx context.Context, sp *StudyPlan) error
	GetStudyPlanByID(ctx context.Context, id uuid.UUID) (*StudyPlanDetail, error)
	GetStudyPlanByStudentAndPeriod(ctx context.Context, studentID, periodID uuid.UUID) (*StudyPlanDetail, error)
	ListStudyPlans(ctx context.Context, filter StudyPlanFilter, p pagination.Params) ([]StudyPlanDetail, int, error)

	// Available classes browsing
	ListAvailableClasses(ctx context.Context, filter AvailableClassFilter, p pagination.Params) ([]AvailableClass, int, error)

	// Enrollments
	GetEnrollmentsByStudyPlan(ctx context.Context, studyPlanID uuid.UUID) ([]EnrollmentDetail, error)
	GetEnrollmentByID(ctx context.Context, enrollmentID uuid.UUID) (*EnrollmentDetail, error)

	// Transactional operations with row locking
	EnrollClassTx(ctx context.Context, studyPlanID, studentID, classID uuid.UUID, bypassOwnership bool) (*EnrollmentDetail, error)
	DropEnrollmentTx(ctx context.Context, studyPlanID, studentID, enrollmentID uuid.UUID, bypassOwnership bool) error
}

type pgRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*students.Student, error) {
	query := `
		SELECT s.id, s.user_id, s.study_program_id, s.nim, s.name, s.email, s.phone, s.gender,
		       s.birth_date, s.admission_year, s.status, s.created_at, s.updated_at,
		       sp.code as study_program_code, sp.name as study_program_name, f.name as faculty_name
		FROM students s
		JOIN study_programs sp ON sp.id = s.study_program_id
		JOIN faculties f ON f.id = sp.faculty_id
		WHERE s.user_id = $1
	`
	var s students.Student
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&s.ID, &s.UserID, &s.StudyProgramID, &s.NIM, &s.Name, &s.Email, &s.Phone, &s.Gender,
		&s.BirthDate, &s.AdmissionYear, &s.Status, &s.CreatedAt, &s.UpdatedAt,
		&s.StudyProgramCode, &s.StudyProgramName, &s.FacultyName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStudentNotFound
		}
		return nil, fmt.Errorf("failed to query student by user id: %w", err)
	}
	return &s, nil
}

func (r *pgRepository) GetStudentByID(ctx context.Context, studentID uuid.UUID) (*students.Student, error) {
	query := `
		SELECT s.id, s.user_id, s.study_program_id, s.nim, s.name, s.email, s.phone, s.gender,
		       s.birth_date, s.admission_year, s.status, s.created_at, s.updated_at,
		       sp.code as study_program_code, sp.name as study_program_name, f.name as faculty_name
		FROM students s
		JOIN study_programs sp ON sp.id = s.study_program_id
		JOIN faculties f ON f.id = sp.faculty_id
		WHERE s.id = $1
	`
	var s students.Student
	err := r.pool.QueryRow(ctx, query, studentID).Scan(
		&s.ID, &s.UserID, &s.StudyProgramID, &s.NIM, &s.Name, &s.Email, &s.Phone, &s.Gender,
		&s.BirthDate, &s.AdmissionYear, &s.Status, &s.CreatedAt, &s.UpdatedAt,
		&s.StudyProgramCode, &s.StudyProgramName, &s.FacultyName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStudentNotFound
		}
		return nil, fmt.Errorf("failed to query student by id: %w", err)
	}
	return &s, nil
}

func (r *pgRepository) CheckAcademicPeriodActive(ctx context.Context, periodID uuid.UUID) (bool, bool, error) {
	query := `SELECT is_active FROM academic_periods WHERE id = $1`
	var isActive bool
	err := r.pool.QueryRow(ctx, query, periodID).Scan(&isActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, nil
		}
		return false, false, fmt.Errorf("failed to check academic period: %w", err)
	}
	return true, isActive, nil
}

func (r *pgRepository) GetActiveAcademicPeriodID(ctx context.Context) (uuid.UUID, error) {
	query := `SELECT id FROM academic_periods WHERE is_active = true LIMIT 1`
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, query).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrAcademicPeriodNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to get active academic period: %w", err)
	}
	return id, nil
}

func (r *pgRepository) CreateStudyPlan(ctx context.Context, sp *StudyPlan) error {
	query := `
		INSERT INTO study_plans (student_id, academic_period_id, total_credits, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		sp.StudentID, sp.AcademicPeriodID, sp.TotalCredits, sp.Status, sp.CreatedAt, sp.UpdatedAt,
	).Scan(&sp.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return ErrDuplicateStudyPlan
			}
			if pgErr.Code == pgerrcode.ForeignKeyViolation {
				if strings.Contains(pgErr.ConstraintName, "student") {
					return ErrStudentNotFound
				}
				if strings.Contains(pgErr.ConstraintName, "academic_period") {
					return ErrAcademicPeriodNotFound
				}
			}
		}
		return fmt.Errorf("failed to insert study plan: %w", err)
	}
	return nil
}

func (r *pgRepository) GetStudyPlanByID(ctx context.Context, id uuid.UUID) (*StudyPlanDetail, error) {
	query := `
		SELECT sp.id, sp.student_id, sp.academic_period_id, sp.total_credits, sp.status, sp.created_at, sp.updated_at,
		       s.nim, s.name, s.study_program_id, prog.code, prog.name,
		       ap.code, ap.name
		FROM study_plans sp
		JOIN students s ON s.id = sp.student_id
		JOIN study_programs prog ON prog.id = s.study_program_id
		JOIN academic_periods ap ON ap.id = sp.academic_period_id
		WHERE sp.id = $1
	`
	var d StudyPlanDetail
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.StudentID, &d.AcademicPeriodID, &d.TotalCredits, &d.Status, &d.CreatedAt, &d.UpdatedAt,
		&d.StudentNIM, &d.StudentName, &d.StudyProgramID, &d.StudyProgramCode, &d.StudyProgramName,
		&d.AcademicPeriodCode, &d.AcademicPeriodName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStudyPlanNotFound
		}
		return nil, fmt.Errorf("failed to query study plan by id: %w", err)
	}

	enrollments, err := r.GetEnrollmentsByStudyPlan(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	d.Enrollments = enrollments
	return &d, nil
}

func (r *pgRepository) GetStudyPlanByStudentAndPeriod(ctx context.Context, studentID, periodID uuid.UUID) (*StudyPlanDetail, error) {
	query := `
		SELECT sp.id, sp.student_id, sp.academic_period_id, sp.total_credits, sp.status, sp.created_at, sp.updated_at,
		       s.nim, s.name, s.study_program_id, prog.code, prog.name,
		       ap.code, ap.name
		FROM study_plans sp
		JOIN students s ON s.id = sp.student_id
		JOIN study_programs prog ON prog.id = s.study_program_id
		JOIN academic_periods ap ON ap.id = sp.academic_period_id
		WHERE sp.student_id = $1 AND sp.academic_period_id = $2
	`
	var d StudyPlanDetail
	err := r.pool.QueryRow(ctx, query, studentID, periodID).Scan(
		&d.ID, &d.StudentID, &d.AcademicPeriodID, &d.TotalCredits, &d.Status, &d.CreatedAt, &d.UpdatedAt,
		&d.StudentNIM, &d.StudentName, &d.StudyProgramID, &d.StudyProgramCode, &d.StudyProgramName,
		&d.AcademicPeriodCode, &d.AcademicPeriodName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStudyPlanNotFound
		}
		return nil, fmt.Errorf("failed to query study plan by student and period: %w", err)
	}

	enrollments, err := r.GetEnrollmentsByStudyPlan(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	d.Enrollments = enrollments
	return &d, nil
}

func (r *pgRepository) ListStudyPlans(ctx context.Context, filter StudyPlanFilter, p pagination.Params) ([]StudyPlanDetail, int, error) {
	whereClauses := []string{"1=1"}
	args := []any{}
	argIdx := 1

	if filter.StudentID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("sp.student_id = $%d", argIdx))
		args = append(args, *filter.StudentID)
		argIdx++
	}
	if filter.AcademicPeriodID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("sp.academic_period_id = $%d", argIdx))
		args = append(args, *filter.AcademicPeriodID)
		argIdx++
	}
	if filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("sp.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM study_plans sp WHERE %s", whereSQL)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count study plans: %w", err)
	}

	limit := p.Limit
	offset := p.Offset()
	selectQuery := fmt.Sprintf(`
		SELECT sp.id, sp.student_id, sp.academic_period_id, sp.total_credits, sp.status, sp.created_at, sp.updated_at,
		       s.nim, s.name, s.study_program_id, prog.code, prog.name,
		       ap.code, ap.name
		FROM study_plans sp
		JOIN students s ON s.id = sp.student_id
		JOIN study_programs prog ON prog.id = s.study_program_id
		JOIN academic_periods ap ON ap.id = sp.academic_period_id
		WHERE %s
		ORDER BY sp.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	argsWithPaging := append(args, limit, offset)
	rows, err := r.pool.Query(ctx, selectQuery, argsWithPaging...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query study plans: %w", err)
	}
	defer rows.Close()

	var list []StudyPlanDetail
	for rows.Next() {
		var d StudyPlanDetail
		if err := rows.Scan(
			&d.ID, &d.StudentID, &d.AcademicPeriodID, &d.TotalCredits, &d.Status, &d.CreatedAt, &d.UpdatedAt,
			&d.StudentNIM, &d.StudentName, &d.StudyProgramID, &d.StudyProgramCode, &d.StudyProgramName,
			&d.AcademicPeriodCode, &d.AcademicPeriodName,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan study plan row: %w", err)
		}
		list = append(list, d)
	}

	return list, total, nil
}

func (r *pgRepository) ListAvailableClasses(ctx context.Context, filter AvailableClassFilter, p pagination.Params) ([]AvailableClass, int, error) {
	whereClauses := []string{"c.status = 'OPEN'", "co.status = 'OPEN'"}
	args := []any{}
	argIdx := 1

	if filter.AcademicPeriodID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("co.academic_period_id = $%d", argIdx))
		args = append(args, *filter.AcademicPeriodID)
		argIdx++
	}
	if filter.StudyProgramID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("co.study_program_id = $%d", argIdx))
		args = append(args, *filter.StudyProgramID)
		argIdx++
	}
	if filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(cr.code ILIKE $%d OR cr.name ILIKE $%d OR c.code ILIKE $%d OR c.name ILIKE $%d)", argIdx, argIdx, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM classes c
		JOIN course_offerings co ON co.id = c.course_offering_id
		JOIN courses cr ON cr.id = co.course_id
		WHERE %s
	`, whereSQL)

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count available classes: %w", err)
	}

	limit := p.Limit
	offset := p.Offset()
	selectQuery := fmt.Sprintf(`
		SELECT c.id, c.code, c.name, c.capacity, c.status,
		       COALESCE((SELECT COUNT(*) FROM enrollments e WHERE e.class_id = c.id AND e.status = 'ENROLLED'), 0) as enrolled_count,
		       co.id, co.status,
		       cr.id, cr.code, cr.name, cr.credits,
		       ap.id, ap.code,
		       sp.id, sp.name
		FROM classes c
		JOIN course_offerings co ON co.id = c.course_offering_id
		JOIN courses cr ON cr.id = co.course_id
		JOIN academic_periods ap ON ap.id = co.academic_period_id
		JOIN study_programs sp ON sp.id = co.study_program_id
		WHERE %s
		ORDER BY cr.code ASC, c.code ASC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	argsWithPaging := append(args, limit, offset)
	rows, err := r.pool.Query(ctx, selectQuery, argsWithPaging...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list available classes: %w", err)
	}
	defer rows.Close()

	var classesList []AvailableClass
	for rows.Next() {
		var ac AvailableClass
		if err := rows.Scan(
			&ac.ClassID, &ac.ClassCode, &ac.ClassName, &ac.Capacity, &ac.ClassStatus,
			&ac.EnrolledCount,
			&ac.CourseOfferingID, &ac.OfferingStatus,
			&ac.CourseID, &ac.CourseCode, &ac.CourseName, &ac.CourseCredits,
			&ac.AcademicPeriodID, &ac.AcademicPeriodCode,
			&ac.StudyProgramID, &ac.StudyProgramName,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan available class: %w", err)
		}
		ac.AvailableSeats = ac.Capacity - ac.EnrolledCount
		if ac.AvailableSeats < 0 {
			ac.AvailableSeats = 0
		}
		classesList = append(classesList, ac)
	}

	// Enrich with lecturers and schedules
	for i := range classesList {
		lecturers, err := r.getClassLecturers(ctx, classesList[i].ClassID)
		if err != nil {
			return nil, 0, err
		}
		classesList[i].Lecturers = lecturers

		schedules, err := r.getClassSchedules(ctx, classesList[i].ClassID)
		if err != nil {
			return nil, 0, err
		}
		classesList[i].Schedules = schedules
	}

	return classesList, total, nil
}

func (r *pgRepository) GetEnrollmentsByStudyPlan(ctx context.Context, studyPlanID uuid.UUID) ([]EnrollmentDetail, error) {
	query := `
		SELECT e.id, e.study_plan_id, e.student_id, e.class_id, e.status, e.enrolled_at, e.dropped_at, e.created_at, e.updated_at,
		       c.code, c.name,
		       cr.id, cr.code, cr.name, cr.credits
		FROM enrollments e
		JOIN classes c ON c.id = e.class_id
		JOIN course_offerings co ON co.id = c.course_offering_id
		JOIN courses cr ON cr.id = co.course_id
		WHERE e.study_plan_id = $1
		ORDER BY cr.code ASC, c.code ASC
	`
	rows, err := r.pool.Query(ctx, query, studyPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to query enrollments: %w", err)
	}
	defer rows.Close()

	var enrollments []EnrollmentDetail
	for rows.Next() {
		var ed EnrollmentDetail
		if err := rows.Scan(
			&ed.ID, &ed.StudyPlanID, &ed.StudentID, &ed.ClassID, &ed.Status, &ed.EnrolledAt, &ed.DroppedAt, &ed.CreatedAt, &ed.UpdatedAt,
			&ed.ClassCode, &ed.ClassName,
			&ed.CourseID, &ed.CourseCode, &ed.CourseName, &ed.CourseCredits,
		); err != nil {
			return nil, fmt.Errorf("failed to scan enrollment: %w", err)
		}
		enrollments = append(enrollments, ed)
	}

	// Enrich with lecturers and schedules
	for i := range enrollments {
		lecturers, err := r.getClassLecturers(ctx, enrollments[i].ClassID)
		if err != nil {
			return nil, err
		}
		enrollments[i].Lecturers = lecturers

		schedules, err := r.getClassSchedules(ctx, enrollments[i].ClassID)
		if err != nil {
			return nil, err
		}
		enrollments[i].Schedules = schedules
	}

	return enrollments, nil
}

func (r *pgRepository) GetEnrollmentByID(ctx context.Context, enrollmentID uuid.UUID) (*EnrollmentDetail, error) {
	query := `
		SELECT e.id, e.study_plan_id, e.student_id, e.class_id, e.status, e.enrolled_at, e.dropped_at, e.created_at, e.updated_at,
		       c.code, c.name,
		       cr.id, cr.code, cr.name, cr.credits
		FROM enrollments e
		JOIN classes c ON c.id = e.class_id
		JOIN course_offerings co ON co.id = c.course_offering_id
		JOIN courses cr ON cr.id = co.course_id
		WHERE e.id = $1
	`
	var ed EnrollmentDetail
	err := r.pool.QueryRow(ctx, query, enrollmentID).Scan(
		&ed.ID, &ed.StudyPlanID, &ed.StudentID, &ed.ClassID, &ed.Status, &ed.EnrolledAt, &ed.DroppedAt, &ed.CreatedAt, &ed.UpdatedAt,
		&ed.ClassCode, &ed.ClassName,
		&ed.CourseID, &ed.CourseCode, &ed.CourseName, &ed.CourseCredits,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEnrollmentNotFound
		}
		return nil, fmt.Errorf("failed to query enrollment: %w", err)
	}

	lecturers, err := r.getClassLecturers(ctx, ed.ClassID)
	if err != nil {
		return nil, err
	}
	ed.Lecturers = lecturers

	schedules, err := r.getClassSchedules(ctx, ed.ClassID)
	if err != nil {
		return nil, err
	}
	ed.Schedules = schedules

	return &ed, nil
}

func (r *pgRepository) EnrollClassTx(ctx context.Context, studyPlanID, studentID, classID uuid.UUID, bypassOwnership bool) (*EnrollmentDetail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Lock and inspect Class + Course Offering + Course
	var classCapacity int
	var classStatus, offeringStatus, courseCode, courseName, classCode, className string
	var courseID, courseOfferingID, periodID uuid.UUID
	var courseCredits int

	classQuery := `
		SELECT c.id, c.code, c.name, c.capacity, c.status,
		       co.id, co.status, co.academic_period_id,
		       cr.id, cr.code, cr.name, cr.credits
		FROM classes c
		JOIN course_offerings co ON co.id = c.course_offering_id
		JOIN courses cr ON cr.id = co.course_id
		WHERE c.id = $1
		FOR UPDATE OF c
	`
	err = tx.QueryRow(ctx, classQuery, classID).Scan(
		&classID, &classCode, &className, &classCapacity, &classStatus,
		&courseOfferingID, &offeringStatus, &periodID,
		&courseID, &courseCode, &courseName, &courseCredits,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrClassNotFound
		}
		return nil, fmt.Errorf("failed to lock class: %w", err)
	}

	if classStatus != "OPEN" {
		return nil, ErrClassNotOpen
	}
	if offeringStatus != "OPEN" {
		return nil, ErrOfferingNotOpen
	}

	// 2. Lock and inspect Study Plan
	var spStudentID, spPeriodID uuid.UUID
	var spTotalCredits int
	var spStatus string

	spQuery := `
		SELECT student_id, academic_period_id, total_credits, status
		FROM study_plans
		WHERE id = $1
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, spQuery, studyPlanID).Scan(
		&spStudentID, &spPeriodID, &spTotalCredits, &spStatus,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStudyPlanNotFound
		}
		return nil, fmt.Errorf("failed to lock study plan: %w", err)
	}

	if !bypassOwnership && spStudentID != studentID {
		return nil, ErrUnauthorizedStudyPlan
	}

	if spStatus != "ACTIVE" && spStatus != "DRAFT" {
		return nil, ErrStudyPlanNotActive
	}

	if spPeriodID != periodID {
		return nil, ErrPeriodMismatch
	}

	// 3. Atomically check class capacity under lock
	var activeCount int
	countQuery := `SELECT COUNT(*) FROM enrollments WHERE class_id = $1 AND status = 'ENROLLED'`
	if err := tx.QueryRow(ctx, countQuery, classID).Scan(&activeCount); err != nil {
		return nil, fmt.Errorf("failed to check class capacity: %w", err)
	}
	if activeCount >= classCapacity {
		return nil, ErrClassCapacityFull
	}

	// 4. Validate Maximum SKS per semester (Max 24 SKS)
	if spTotalCredits+courseCredits > 24 {
		return nil, ErrMaxCreditsExceeded
	}

	// 5. Prevent duplicate active enrollment in this exact class
	var alreadyInClass bool
	dupClassQuery := `
		SELECT EXISTS(
			SELECT 1 FROM enrollments
			WHERE student_id = $1 AND class_id = $2 AND status = 'ENROLLED'
		)
	`
	if err := tx.QueryRow(ctx, dupClassQuery, spStudentID, classID).Scan(&alreadyInClass); err != nil {
		return nil, fmt.Errorf("failed to check duplicate class enrollment: %w", err)
	}
	if alreadyInClass {
		return nil, ErrAlreadyEnrolledInClass
	}

	// 6. Prevent enrolling in two different classes for the SAME course in this study plan
	var alreadyInCourse bool
	dupCourseQuery := `
		SELECT EXISTS(
			SELECT 1 FROM enrollments e
			JOIN classes c ON c.id = e.class_id
			JOIN course_offerings co ON co.id = c.course_offering_id
			WHERE e.study_plan_id = $1 AND e.status = 'ENROLLED' AND co.course_id = $2
		)
	`
	if err := tx.QueryRow(ctx, dupCourseQuery, studyPlanID, courseID).Scan(&alreadyInCourse); err != nil {
		return nil, fmt.Errorf("failed to check duplicate course in study plan: %w", err)
	}
	if alreadyInCourse {
		return nil, ErrAlreadyEnrolledInCourse
	}

	// 7. Check Schedule Conflict
	// Query target class schedules
	targetSchedules, err := r.getClassSchedulesTx(ctx, tx, classID)
	if err != nil {
		return nil, err
	}

	if len(targetSchedules) > 0 {
		// Query existing active schedules in this study plan
		existingSchedulesQuery := `
			SELECT cs.day_of_week, cs.start_time::text, cs.end_time::text, c.name, cr.name
			FROM enrollments e
			JOIN classes c ON c.id = e.class_id
			JOIN course_offerings co ON co.id = c.course_offering_id
			JOIN courses cr ON cr.id = co.course_id
			JOIN class_schedules cs ON cs.class_id = c.id
			WHERE e.study_plan_id = $1 AND e.status = 'ENROLLED'
		`
		rows, err := tx.Query(ctx, existingSchedulesQuery, studyPlanID)
		if err != nil {
			return nil, fmt.Errorf("failed to query existing schedules: %w", err)
		}
		defer rows.Close()

		type scheduleRecord struct {
			dayOfWeek  int
			startTime  string
			endTime    string
			className  string
			courseName string
		}
		var existing []scheduleRecord
		for rows.Next() {
			var sr scheduleRecord
			if err := rows.Scan(&sr.dayOfWeek, &sr.startTime, &sr.endTime, &sr.className, &sr.courseName); err != nil {
				return nil, fmt.Errorf("failed to scan existing schedule: %w", err)
			}
			existing = append(existing, sr)
		}
		rows.Close()

		for _, ts := range targetSchedules {
			for _, es := range existing {
				if ts.DayOfWeek == es.dayOfWeek {
					// Half-open interval [start, end) overlap: startA < endB AND startB < endA
					if isTimeOverlap(ts.StartTime, ts.EndTime, es.startTime, es.endTime) {
						return nil, appErrors.ErrConflict(fmt.Sprintf(
							"Jadwal kelas bertabrakan dengan kelas '%s (%s)' pada hari %d (%s - %s)",
							es.courseName, es.className, es.dayOfWeek, es.startTime, es.endTime,
						))
					}
				}
			}
		}
	}

	// 8. Insert Enrollment record
	var enrollmentID uuid.UUID
	insertQuery := `
		INSERT INTO enrollments (study_plan_id, student_id, class_id, status, enrolled_at, created_at, updated_at)
		VALUES ($1, $2, $3, 'ENROLLED', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`
	err = tx.QueryRow(ctx, insertQuery, studyPlanID, spStudentID, classID).Scan(&enrollmentID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return nil, ErrAlreadyEnrolledInClass
		}
		return nil, fmt.Errorf("failed to insert enrollment: %w", err)
	}

	// 9. Update study plan total credits
	updateCreditsQuery := `
		UPDATE study_plans
		SET total_credits = total_credits + $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	if _, err := tx.Exec(ctx, updateCreditsQuery, courseCredits, studyPlanID); err != nil {
		return nil, fmt.Errorf("failed to update study plan credits: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit enrollment: %w", err)
	}

	return r.GetEnrollmentByID(ctx, enrollmentID)
}

func (r *pgRepository) DropEnrollmentTx(ctx context.Context, studyPlanID, studentID, enrollmentID uuid.UUID, bypassOwnership bool) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Lock Study Plan
	var spStudentID uuid.UUID
	var spStatus string
	spQuery := `
		SELECT student_id, status
		FROM study_plans
		WHERE id = $1
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, spQuery, studyPlanID).Scan(&spStudentID, &spStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrStudyPlanNotFound
		}
		return fmt.Errorf("failed to lock study plan: %w", err)
	}

	if !bypassOwnership && spStudentID != studentID {
		return ErrUnauthorizedStudyPlan
	}

	if spStatus != "ACTIVE" && spStatus != "DRAFT" {
		return ErrStudyPlanNotActive
	}

	// 2. Lock Enrollment
	var currentStatus string
	var courseCredits int
	enrollQuery := `
		SELECT e.status, cr.credits
		FROM enrollments e
		JOIN classes c ON c.id = e.class_id
		JOIN course_offerings co ON co.id = c.course_offering_id
		JOIN courses cr ON cr.id = co.course_id
		WHERE e.id = $1 AND e.study_plan_id = $2
		FOR UPDATE OF e
	`
	err = tx.QueryRow(ctx, enrollQuery, enrollmentID, studyPlanID).Scan(&currentStatus, &courseCredits)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEnrollmentNotFound
		}
		return fmt.Errorf("failed to lock enrollment: %w", err)
	}

	if currentStatus != "ENROLLED" {
		return ErrEnrollmentNotActive
	}

	// 3. Mark enrollment as DROPPED
	updateEnrollQuery := `
		UPDATE enrollments
		SET status = 'DROPPED', dropped_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	if _, err := tx.Exec(ctx, updateEnrollQuery, enrollmentID); err != nil {
		return fmt.Errorf("failed to drop enrollment: %w", err)
	}

	// 4. Decrement study plan total credits
	updateCreditsQuery := `
		UPDATE study_plans
		SET total_credits = GREATEST(0, total_credits - $1), updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	if _, err := tx.Exec(ctx, updateCreditsQuery, courseCredits, studyPlanID); err != nil {
		return fmt.Errorf("failed to update study plan credits: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *pgRepository) getClassLecturers(ctx context.Context, classID uuid.UUID) ([]classes.ClassLecturerDetail, error) {
	query := `
		SELECT cl.id, cl.class_id, cl.lecturer_id, cl.role, cl.status, cl.created_at, cl.updated_at,
		       l.nidn, l.name, l.email
		FROM class_lecturers cl
		JOIN lecturers l ON l.id = cl.lecturer_id
		WHERE cl.class_id = $1 AND cl.status = 'ACTIVE'
		ORDER BY cl.role ASC, l.name ASC
	`
	rows, err := r.pool.Query(ctx, query, classID)
	if err != nil {
		return nil, fmt.Errorf("failed to query class lecturers: %w", err)
	}
	defer rows.Close()

	var lecturers []classes.ClassLecturerDetail
	for rows.Next() {
		var ld classes.ClassLecturerDetail
		if err := rows.Scan(
			&ld.ID, &ld.ClassID, &ld.LecturerID, &ld.Role, &ld.Status, &ld.CreatedAt, &ld.UpdatedAt,
			&ld.LecturerNIDN, &ld.LecturerName, &ld.LecturerEmail,
		); err != nil {
			return nil, fmt.Errorf("failed to scan lecturer detail: %w", err)
		}
		lecturers = append(lecturers, ld)
	}
	return lecturers, nil
}

func (r *pgRepository) getClassSchedules(ctx context.Context, classID uuid.UUID) ([]classes.ClassSchedule, error) {
	query := `
		SELECT id, class_id, day_of_week, start_time::text, end_time::text, room, delivery_mode, created_at, updated_at
		FROM class_schedules
		WHERE class_id = $1
		ORDER BY day_of_week ASC, start_time ASC
	`
	rows, err := r.pool.Query(ctx, query, classID)
	if err != nil {
		return nil, fmt.Errorf("failed to query class schedules: %w", err)
	}
	defer rows.Close()

	var schedules []classes.ClassSchedule
	for rows.Next() {
		var cs classes.ClassSchedule
		if err := rows.Scan(
			&cs.ID, &cs.ClassID, &cs.DayOfWeek, &cs.StartTime, &cs.EndTime, &cs.Room, &cs.DeliveryMode,
			&cs.CreatedAt, &cs.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan class schedule: %w", err)
		}
		schedules = append(schedules, cs)
	}
	return schedules, nil
}

func (r *pgRepository) getClassSchedulesTx(ctx context.Context, tx pgx.Tx, classID uuid.UUID) ([]classes.ClassSchedule, error) {
	query := `
		SELECT id, class_id, day_of_week, start_time::text, end_time::text, room, delivery_mode, created_at, updated_at
		FROM class_schedules
		WHERE class_id = $1
		ORDER BY day_of_week ASC, start_time ASC
	`
	rows, err := tx.Query(ctx, query, classID)
	if err != nil {
		return nil, fmt.Errorf("failed to query class schedules in tx: %w", err)
	}
	defer rows.Close()

	var schedules []classes.ClassSchedule
	for rows.Next() {
		var cs classes.ClassSchedule
		if err := rows.Scan(
			&cs.ID, &cs.ClassID, &cs.DayOfWeek, &cs.StartTime, &cs.EndTime, &cs.Room, &cs.DeliveryMode,
			&cs.CreatedAt, &cs.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan class schedule: %w", err)
		}
		schedules = append(schedules, cs)
	}
	return schedules, nil
}

func isTimeOverlap(s1, e1, s2, e2 string) bool {
	// Format HH:MM or HH:MM:SS
	startA := normalizeTime(s1)
	endA := normalizeTime(e1)
	startB := normalizeTime(s2)
	endB := normalizeTime(e2)

	return startA < endB && startB < endA
}

func normalizeTime(t string) string {
	parts := strings.Split(t, ":")
	if len(parts) >= 2 {
		return fmt.Sprintf("%02s:%02s", parts[0], parts[1])
	}
	return t
}
