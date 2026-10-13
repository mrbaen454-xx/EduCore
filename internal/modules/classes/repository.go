package classes

import (
	"context"
	"errors"
	"fmt"
	"strings"

	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrClassNotFound      = appErrors.ErrNotFound("Kelas perkuliahan tidak ditemukan")
	ErrScheduleNotFound   = appErrors.ErrNotFound("Jadwal perkuliahan tidak ditemukan")
	ErrLecturerNotFound   = appErrors.ErrNotFound("Dosen tidak ditemukan")
	ErrDuplicateClassCode = appErrors.ErrConflict("Kode kelas sudah digunakan pada penawaran ini")
	ErrDuplicateLecturer  = appErrors.ErrConflict("Dosen sudah ditugaskan pada kelas ini")
	ErrScheduleConflict   = appErrors.ErrConflict("Jadwal perkuliahan bertabrakan dengan jadwal lain")
)

type Repository interface {
	// Classes
	CreateClass(ctx context.Context, c *Class) error
	GetClassByID(ctx context.Context, id uuid.UUID) (*ClassDetail, error)
	ListClasses(ctx context.Context, filter ClassFilter, p pagination.Params) ([]ClassDetail, int, error)
	UpdateClass(ctx context.Context, c *Class) error
	UpdateClassStatus(ctx context.Context, id uuid.UUID, status string) error

	// Class Lecturers
	AssignLecturer(ctx context.Context, cl *ClassLecturer) error
	GetClassLecturers(ctx context.Context, classID uuid.UUID) ([]ClassLecturerDetail, error)
	RemoveLecturer(ctx context.Context, classID, lecturerID uuid.UUID) error

	// Class Schedules
	CreateSchedule(ctx context.Context, cs *ClassSchedule) error
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*ClassSchedule, error)
	GetClassSchedules(ctx context.Context, classID uuid.UUID) ([]ClassSchedule, error)
	UpdateSchedule(ctx context.Context, cs *ClassSchedule) error
	DeleteSchedule(ctx context.Context, id uuid.UUID) error

	// Conflict Checks
	CheckClassSelfConflict(ctx context.Context, classID, excludeScheduleID uuid.UUID, dayOfWeek int, startTime, endTime string) (bool, error)
	CheckLecturerScheduleConflict(ctx context.Context, periodID, classID, excludeScheduleID uuid.UUID, dayOfWeek int, startTime, endTime string) (bool, string, error)
	CheckRoomScheduleConflict(ctx context.Context, periodID, classID, excludeScheduleID uuid.UUID, dayOfWeek int, startTime, endTime, room string) (bool, string, error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) CreateClass(ctx context.Context, c *Class) error {
	query := `
		INSERT INTO classes (id, course_offering_id, code, name, capacity, status, remarks, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING created_at, updated_at;
	`
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		c.ID, c.CourseOfferingID, c.Code, c.Name, c.Capacity, c.Status, c.Remarks,
	).Scan(&c.CreatedAt, &c.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return ErrDuplicateClassCode
			}
			if pgErr.Code == "23503" {
				return appErrors.ErrNotFound("Penawaran mata kuliah tidak ditemukan")
			}
		}
		return appErrors.ErrInternal(err)
	}

	return nil
}

func (r *pgRepository) GetClassByID(ctx context.Context, id uuid.UUID) (*ClassDetail, error) {
	query := `
		SELECT 
			cls.id, cls.course_offering_id, cls.code, cls.name, cls.capacity, cls.status, cls.remarks, cls.created_at, cls.updated_at,
			c.code AS course_code, c.name AS course_name, c.credits AS course_credits,
			ap.id AS period_id, ap.code AS period_code, ap.name AS period_name,
			sp.id AS prodi_id, sp.name AS prodi_name
		FROM classes cls
		JOIN course_offerings co ON co.id = cls.course_offering_id
		JOIN courses c ON c.id = co.course_id
		JOIN academic_periods ap ON ap.id = co.academic_period_id
		JOIN study_programs sp ON sp.id = co.study_program_id
		WHERE cls.id = $1;
	`
	var d ClassDetail
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.CourseOfferingID, &d.Code, &d.Name, &d.Capacity, &d.Status, &d.Remarks, &d.CreatedAt, &d.UpdatedAt,
		&d.CourseCode, &d.CourseName, &d.CourseCredits,
		&d.AcademicPeriodID, &d.AcademicPeriodCode, &d.AcademicPeriodName,
		&d.StudyProgramID, &d.StudyProgramName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrClassNotFound
		}
		return nil, appErrors.ErrInternal(err)
	}

	// Fetch lecturers
	lecturers, err := r.GetClassLecturers(ctx, id)
	if err != nil {
		return nil, err
	}
	d.Lecturers = lecturers

	// Fetch schedules
	schedules, err := r.GetClassSchedules(ctx, id)
	if err != nil {
		return nil, err
	}
	d.Schedules = schedules

	return &d, nil
}

func (r *pgRepository) ListClasses(ctx context.Context, filter ClassFilter, p pagination.Params) ([]ClassDetail, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.CourseOfferingID != nil {
		conditions = append(conditions, fmt.Sprintf("cls.course_offering_id = $%d", argIdx))
		args = append(args, *filter.CourseOfferingID)
		argIdx++
	}
	if filter.AcademicPeriodID != nil {
		conditions = append(conditions, fmt.Sprintf("co.academic_period_id = $%d", argIdx))
		args = append(args, *filter.AcademicPeriodID)
		argIdx++
	}
	if filter.StudyProgramID != nil {
		conditions = append(conditions, fmt.Sprintf("co.study_program_id = $%d", argIdx))
		args = append(args, *filter.StudyProgramID)
		argIdx++
	}
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("cls.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(cls.code ILIKE $%d OR cls.name ILIKE $%d OR c.name ILIKE $%d OR c.code ILIKE $%d)", argIdx, argIdx, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT cls.id)
		FROM classes cls
		JOIN course_offerings co ON co.id = cls.course_offering_id
		JOIN courses c ON c.id = co.course_id
		%s;
	`, whereClause)

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, appErrors.ErrInternal(err)
	}

	allowedSorts := map[string]string{
		"code":        "cls.code",
		"name":        "cls.name",
		"capacity":    "cls.capacity",
		"created_at":  "cls.created_at",
		"-code":       "cls.code DESC",
		"-name":       "cls.name DESC",
		"-capacity":   "cls.capacity DESC",
		"-created_at": "cls.created_at DESC",
	}

	orderBy := "cls.code ASC"
	if p.Sort != "" {
		if col, ok := allowedSorts[p.Sort]; ok {
			orderBy = col
		}
	}

	dataQuery := fmt.Sprintf(`
		SELECT 
			cls.id, cls.course_offering_id, cls.code, cls.name, cls.capacity, cls.status, cls.remarks, cls.created_at, cls.updated_at,
			c.code AS course_code, c.name AS course_name, c.credits AS course_credits,
			ap.id AS period_id, ap.code AS period_code, ap.name AS period_name,
			sp.id AS prodi_id, sp.name AS prodi_name
		FROM classes cls
		JOIN course_offerings co ON co.id = cls.course_offering_id
		JOIN courses c ON c.id = co.course_id
		JOIN academic_periods ap ON ap.id = co.academic_period_id
		JOIN study_programs sp ON sp.id = co.study_program_id
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d;
	`, whereClause, orderBy, argIdx, argIdx+1)

	args = append(args, p.Limit, p.Offset())

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, appErrors.ErrInternal(err)
	}
	defer rows.Close()

	var result []ClassDetail
	for rows.Next() {
		var d ClassDetail
		if err := rows.Scan(
			&d.ID, &d.CourseOfferingID, &d.Code, &d.Name, &d.Capacity, &d.Status, &d.Remarks, &d.CreatedAt, &d.UpdatedAt,
			&d.CourseCode, &d.CourseName, &d.CourseCredits,
			&d.AcademicPeriodID, &d.AcademicPeriodCode, &d.AcademicPeriodName,
			&d.StudyProgramID, &d.StudyProgramName,
		); err != nil {
			return nil, 0, appErrors.ErrInternal(err)
		}
		result = append(result, d)
	}

	return result, total, nil
}

func (r *pgRepository) UpdateClass(ctx context.Context, c *Class) error {
	query := `
		UPDATE classes
		SET name = $2, capacity = $3, remarks = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at;
	`
	err := r.pool.QueryRow(ctx, query,
		c.ID, c.Name, c.Capacity, c.Remarks,
	).Scan(&c.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrClassNotFound
		}
		return appErrors.ErrInternal(err)
	}
	return nil
}

func (r *pgRepository) UpdateClassStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE classes
		SET status = $2, updated_at = NOW()
		WHERE id = $1;
	`
	cmdTag, err := r.pool.Exec(ctx, query, id, status)
	if err != nil {
		return appErrors.ErrInternal(err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrClassNotFound
	}
	return nil
}

// Class Lecturers

func (r *pgRepository) AssignLecturer(ctx context.Context, cl *ClassLecturer) error {
	query := `
		INSERT INTO class_lecturers (id, class_id, lecturer_id, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING created_at, updated_at;
	`
	if cl.ID == uuid.Nil {
		cl.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		cl.ID, cl.ClassID, cl.LecturerID, cl.Role, cl.Status,
	).Scan(&cl.CreatedAt, &cl.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return ErrDuplicateLecturer
			}
			if pgErr.Code == "23503" {
				return ErrLecturerNotFound
			}
		}
		return appErrors.ErrInternal(err)
	}
	return nil
}

func (r *pgRepository) GetClassLecturers(ctx context.Context, classID uuid.UUID) ([]ClassLecturerDetail, error) {
	query := `
		SELECT 
			cl.id, cl.class_id, cl.lecturer_id, cl.role, cl.status, cl.created_at, cl.updated_at,
			l.nidn, l.name, l.email
		FROM class_lecturers cl
		JOIN lecturers l ON l.id = cl.lecturer_id
		WHERE cl.class_id = $1
		ORDER BY cl.created_at ASC;
	`
	rows, err := r.pool.Query(ctx, query, classID)
	if err != nil {
		return nil, appErrors.ErrInternal(err)
	}
	defer rows.Close()

	var result []ClassLecturerDetail
	for rows.Next() {
		var d ClassLecturerDetail
		if err := rows.Scan(
			&d.ID, &d.ClassID, &d.LecturerID, &d.Role, &d.Status, &d.CreatedAt, &d.UpdatedAt,
			&d.LecturerNIDN, &d.LecturerName, &d.LecturerEmail,
		); err != nil {
			return nil, appErrors.ErrInternal(err)
		}
		result = append(result, d)
	}
	return result, nil
}

func (r *pgRepository) RemoveLecturer(ctx context.Context, classID, lecturerID uuid.UUID) error {
	query := `
		DELETE FROM class_lecturers
		WHERE class_id = $1 AND lecturer_id = $2;
	`
	cmdTag, err := r.pool.Exec(ctx, query, classID, lecturerID)
	if err != nil {
		return appErrors.ErrInternal(err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrLecturerNotFound
	}
	return nil
}

// Class Schedules

func (r *pgRepository) CreateSchedule(ctx context.Context, cs *ClassSchedule) error {
	query := `
		INSERT INTO class_schedules (id, class_id, day_of_week, start_time, end_time, room, delivery_mode, created_at, updated_at)
		VALUES ($1, $2, $3, $4::time, $5::time, $6, $7, NOW(), NOW())
		RETURNING created_at, updated_at;
	`
	if cs.ID == uuid.Nil {
		cs.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		cs.ID, cs.ClassID, cs.DayOfWeek, cs.StartTime, cs.EndTime, cs.Room, cs.DeliveryMode,
	).Scan(&cs.CreatedAt, &cs.UpdatedAt)

	if err != nil {
		return appErrors.ErrInternal(err)
	}
	return nil
}

func (r *pgRepository) GetScheduleByID(ctx context.Context, id uuid.UUID) (*ClassSchedule, error) {
	query := `
		SELECT id, class_id, day_of_week, to_char(start_time, 'HH24:MI:SS'), to_char(end_time, 'HH24:MI:SS'), room, delivery_mode, created_at, updated_at
		FROM class_schedules
		WHERE id = $1;
	`
	var cs ClassSchedule
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&cs.ID, &cs.ClassID, &cs.DayOfWeek, &cs.StartTime, &cs.EndTime, &cs.Room, &cs.DeliveryMode, &cs.CreatedAt, &cs.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScheduleNotFound
		}
		return nil, appErrors.ErrInternal(err)
	}
	return &cs, nil
}

func (r *pgRepository) GetClassSchedules(ctx context.Context, classID uuid.UUID) ([]ClassSchedule, error) {
	query := `
		SELECT id, class_id, day_of_week, to_char(start_time, 'HH24:MI:SS'), to_char(end_time, 'HH24:MI:SS'), room, delivery_mode, created_at, updated_at
		FROM class_schedules
		WHERE class_id = $1
		ORDER BY day_of_week ASC, start_time ASC;
	`
	rows, err := r.pool.Query(ctx, query, classID)
	if err != nil {
		return nil, appErrors.ErrInternal(err)
	}
	defer rows.Close()

	var result []ClassSchedule
	for rows.Next() {
		var cs ClassSchedule
		if err := rows.Scan(
			&cs.ID, &cs.ClassID, &cs.DayOfWeek, &cs.StartTime, &cs.EndTime, &cs.Room, &cs.DeliveryMode, &cs.CreatedAt, &cs.UpdatedAt,
		); err != nil {
			return nil, appErrors.ErrInternal(err)
		}
		result = append(result, cs)
	}
	return result, nil
}

func (r *pgRepository) UpdateSchedule(ctx context.Context, cs *ClassSchedule) error {
	query := `
		UPDATE class_schedules
		SET day_of_week = $2, start_time = $3::time, end_time = $4::time, room = $5, delivery_mode = $6, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at;
	`
	err := r.pool.QueryRow(ctx, query,
		cs.ID, cs.DayOfWeek, cs.StartTime, cs.EndTime, cs.Room, cs.DeliveryMode,
	).Scan(&cs.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrScheduleNotFound
		}
		return appErrors.ErrInternal(err)
	}
	return nil
}

func (r *pgRepository) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM class_schedules WHERE id = $1;`
	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return appErrors.ErrInternal(err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrScheduleNotFound
	}
	return nil
}

// Conflict Checks

func (r *pgRepository) CheckClassSelfConflict(ctx context.Context, classID, excludeScheduleID uuid.UUID, dayOfWeek int, startTime, endTime string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM class_schedules
			WHERE class_id = $1
			  AND id != $2
			  AND day_of_week = $3
			  AND start_time < $5::time AND end_time > $4::time
		);
	`
	var conflict bool
	err := r.pool.QueryRow(ctx, query, classID, excludeScheduleID, dayOfWeek, startTime, endTime).Scan(&conflict)
	if err != nil {
		return false, appErrors.ErrInternal(err)
	}
	return conflict, nil
}

func (r *pgRepository) CheckLecturerScheduleConflict(ctx context.Context, periodID, classID, excludeScheduleID uuid.UUID, dayOfWeek int, startTime, endTime string) (bool, string, error) {
	// Find lecturers assigned to this class
	// Check if any of those lecturers has another schedule in the same academic period at the same day/time
	query := `
		SELECT l.name, c.name AS conflicting_class
		FROM class_lecturers cl
		JOIN lecturers l ON l.id = cl.lecturer_id
		JOIN classes other_cls ON other_cls.id != $2
		JOIN class_lecturers other_cl ON other_cl.class_id = other_cls.id AND other_cl.lecturer_id = cl.lecturer_id
		JOIN course_offerings other_co ON other_co.id = other_cls.course_offering_id
		JOIN class_schedules other_cs ON other_cs.class_id = other_cls.id
		JOIN classes c ON c.id = other_cls.id
		WHERE cl.class_id = $2
		  AND other_co.academic_period_id = $1
		  AND other_cs.id != $3
		  AND other_cs.day_of_week = $4
		  AND other_cs.start_time < $6::time AND other_cs.end_time > $5::time
		  AND other_cls.status != 'CANCELLED'
		LIMIT 1;
	`
	var lecturerName, conflictingClass string
	err := r.pool.QueryRow(ctx, query, periodID, classID, excludeScheduleID, dayOfWeek, startTime, endTime).Scan(&lecturerName, &conflictingClass)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil
		}
		return false, "", appErrors.ErrInternal(err)
	}
	return true, fmt.Sprintf("Dosen %s memiliki jadwal mengajar di kelas %s pada waktu yang sama", lecturerName, conflictingClass), nil
}

func (r *pgRepository) CheckRoomScheduleConflict(ctx context.Context, periodID, classID, excludeScheduleID uuid.UUID, dayOfWeek int, startTime, endTime, room string) (bool, string, error) {
	if strings.TrimSpace(room) == "" {
		return false, "", nil
	}

	query := `
		SELECT c.name AS conflicting_class
		FROM class_schedules other_cs
		JOIN classes other_cls ON other_cls.id = other_cs.class_id
		JOIN course_offerings other_co ON other_co.id = other_cls.course_offering_id
		JOIN classes c ON c.id = other_cls.id
		WHERE other_co.academic_period_id = $1
		  AND other_cs.id != $3
		  AND other_cs.class_id != $2
		  AND other_cs.day_of_week = $4
		  AND LOWER(TRIM(other_cs.room)) = LOWER(TRIM($7))
		  AND other_cs.start_time < $6::time AND other_cs.end_time > $5::time
		  AND other_cls.status != 'CANCELLED'
		LIMIT 1;
	`
	var conflictingClass string
	err := r.pool.QueryRow(ctx, query, periodID, classID, excludeScheduleID, dayOfWeek, startTime, endTime, room).Scan(&conflictingClass)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil
		}
		return false, "", appErrors.ErrInternal(err)
	}
	return true, fmt.Sprintf("Ruangan %s sudah digunakan oleh kelas %s pada waktu yang sama", room, conflictingClass), nil
}
