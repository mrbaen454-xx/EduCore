package course_offerings

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
	ErrNotFound          = appErrors.ErrNotFound("Penawaran mata kuliah tidak ditemukan")
	ErrDuplicateOffering = appErrors.ErrConflict("Mata kuliah sudah ditawarkan pada periode akademik dan program studi ini")
)

type Repository interface {
	Create(ctx context.Context, co *CourseOffering) error
	GetByID(ctx context.Context, id uuid.UUID) (*CourseOfferingDetail, error)
	List(ctx context.Context, filter CourseOfferingFilter, p pagination.Params) ([]CourseOfferingDetail, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	CheckForeignEntities(ctx context.Context, periodID, courseID, studyProgramID uuid.UUID, curriculumID *uuid.UUID) (periodActive, courseActive, prodiActive bool, currValid bool, err error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) CheckForeignEntities(ctx context.Context, periodID, courseID, studyProgramID uuid.UUID, curriculumID *uuid.UUID) (bool, bool, bool, bool, error) {
	var periodActive, courseActive, prodiActive bool
	currValid := true

	// Check academic period
	err := r.pool.QueryRow(ctx, "SELECT is_active FROM academic_periods WHERE id = $1;", periodID).Scan(&periodActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, false, false, appErrors.ErrNotFound("Periode akademik tidak ditemukan")
		}
		return false, false, false, false, appErrors.ErrInternal(err)
	}

	// Check course
	err = r.pool.QueryRow(ctx, "SELECT is_active FROM courses WHERE id = $1;", courseID).Scan(&courseActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, false, false, appErrors.ErrNotFound("Mata kuliah tidak ditemukan")
		}
		return false, false, false, false, appErrors.ErrInternal(err)
	}

	// Check study program
	err = r.pool.QueryRow(ctx, "SELECT is_active FROM study_programs WHERE id = $1;", studyProgramID).Scan(&prodiActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, false, false, appErrors.ErrNotFound("Program studi tidak ditemukan")
		}
		return false, false, false, false, appErrors.ErrInternal(err)
	}

	// Check curriculum if provided
	if curriculumID != nil {
		var exists bool
		err = r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM curricula WHERE id = $1 AND study_program_id = $2);", *curriculumID, studyProgramID).Scan(&exists)
		if err != nil {
			return false, false, false, false, appErrors.ErrInternal(err)
		}
		if !exists {
			return false, false, false, false, appErrors.ErrNotFound("Kurikulum tidak ditemukan atau tidak sesuai dengan program studi")
		}
	}

	return periodActive, courseActive, prodiActive, currValid, nil
}

func (r *pgRepository) Create(ctx context.Context, co *CourseOffering) error {
	query := `
		INSERT INTO course_offerings (id, academic_period_id, course_id, curriculum_id, study_program_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING created_at, updated_at;
	`
	if co.ID == uuid.Nil {
		co.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		co.ID, co.AcademicPeriodID, co.CourseID, co.CurriculumID, co.StudyProgramID, co.Status,
	).Scan(&co.CreatedAt, &co.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateOffering
		}
		return appErrors.ErrInternal(err)
	}

	return nil
}

func (r *pgRepository) GetByID(ctx context.Context, id uuid.UUID) (*CourseOfferingDetail, error) {
	query := `
		SELECT 
			co.id, co.academic_period_id, co.course_id, co.curriculum_id, co.study_program_id, co.status, co.created_at, co.updated_at,
			ap.code AS period_code, ap.name AS period_name,
			c.code AS course_code, c.name AS course_name, c.credits AS course_credits,
			sp.code AS prodi_code, sp.name AS prodi_name,
			COALESCE(curr.name, '') AS curriculum_name,
			COUNT(cls.id) AS total_classes
		FROM course_offerings co
		JOIN academic_periods ap ON ap.id = co.academic_period_id
		JOIN courses c ON c.id = co.course_id
		JOIN study_programs sp ON sp.id = co.study_program_id
		LEFT JOIN curricula curr ON curr.id = co.curriculum_id
		LEFT JOIN classes cls ON cls.course_offering_id = co.id
		WHERE co.id = $1
		GROUP BY co.id, ap.code, ap.name, c.code, c.name, c.credits, sp.code, sp.name, curr.name;
	`
	var d CourseOfferingDetail
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.AcademicPeriodID, &d.CourseID, &d.CurriculumID, &d.StudyProgramID, &d.Status, &d.CreatedAt, &d.UpdatedAt,
		&d.AcademicPeriodCode, &d.AcademicPeriodName,
		&d.CourseCode, &d.CourseName, &d.CourseCredits,
		&d.StudyProgramCode, &d.StudyProgramName,
		&d.CurriculumName,
		&d.TotalClasses,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, appErrors.ErrInternal(err)
	}
	return &d, nil
}

func (r *pgRepository) List(ctx context.Context, filter CourseOfferingFilter, p pagination.Params) ([]CourseOfferingDetail, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

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
	if filter.CourseID != nil {
		conditions = append(conditions, fmt.Sprintf("co.course_id = $%d", argIdx))
		args = append(args, *filter.CourseID)
		argIdx++
	}
	if filter.CurriculumID != nil {
		conditions = append(conditions, fmt.Sprintf("co.curriculum_id = $%d", argIdx))
		args = append(args, *filter.CurriculumID)
		argIdx++
	}
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("co.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(c.code ILIKE $%d OR c.name ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT co.id)
		FROM course_offerings co
		JOIN courses c ON c.id = co.course_id
		%s;
	`, whereClause)

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, appErrors.ErrInternal(err)
	}

	allowedSorts := map[string]string{
		"course_code":  "c.code",
		"created_at":   "co.created_at",
		"-course_code": "c.code DESC",
		"-created_at":  "co.created_at DESC",
	}

	orderBy := "c.code ASC"
	if p.Sort != "" {
		if col, ok := allowedSorts[p.Sort]; ok {
			orderBy = col
		}
	}

	dataQuery := fmt.Sprintf(`
		SELECT 
			co.id, co.academic_period_id, co.course_id, co.curriculum_id, co.study_program_id, co.status, co.created_at, co.updated_at,
			ap.code AS period_code, ap.name AS period_name,
			c.code AS course_code, c.name AS course_name, c.credits AS course_credits,
			sp.code AS prodi_code, sp.name AS prodi_name,
			COALESCE(curr.name, '') AS curriculum_name,
			COUNT(cls.id) AS total_classes
		FROM course_offerings co
		JOIN academic_periods ap ON ap.id = co.academic_period_id
		JOIN courses c ON c.id = co.course_id
		JOIN study_programs sp ON sp.id = co.study_program_id
		LEFT JOIN curricula curr ON curr.id = co.curriculum_id
		LEFT JOIN classes cls ON cls.course_offering_id = co.id
		%s
		GROUP BY co.id, ap.code, ap.name, c.code, c.name, c.credits, sp.code, sp.name, curr.name
		ORDER BY %s
		LIMIT $%d OFFSET $%d;
	`, whereClause, orderBy, argIdx, argIdx+1)

	args = append(args, p.Limit, p.Offset())

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, appErrors.ErrInternal(err)
	}
	defer rows.Close()

	var result []CourseOfferingDetail
	for rows.Next() {
		var d CourseOfferingDetail
		if err := rows.Scan(
			&d.ID, &d.AcademicPeriodID, &d.CourseID, &d.CurriculumID, &d.StudyProgramID, &d.Status, &d.CreatedAt, &d.UpdatedAt,
			&d.AcademicPeriodCode, &d.AcademicPeriodName,
			&d.CourseCode, &d.CourseName, &d.CourseCredits,
			&d.StudyProgramCode, &d.StudyProgramName,
			&d.CurriculumName,
			&d.TotalClasses,
		); err != nil {
			return nil, 0, appErrors.ErrInternal(err)
		}
		result = append(result, d)
	}

	return result, total, nil
}

func (r *pgRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE course_offerings
		SET status = $2, updated_at = NOW()
		WHERE id = $1;
	`
	cmdTag, err := r.pool.Exec(ctx, query, id, status)
	if err != nil {
		return appErrors.ErrInternal(err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
