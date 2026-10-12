package curricula

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"educore/internal/shared/pagination"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound             = errors.New("curriculum not found")
	ErrDuplicateCode        = errors.New("curriculum code already exists")
	ErrDuplicateCourse      = errors.New("course already associated with curriculum")
	ErrCourseNotFound       = errors.New("course not found in curriculum")
	ErrStudyProgramNotFound = errors.New("study program not found")
)

// Repository defines persistence operations for curricula and curriculum courses.
type Repository interface {
	Create(ctx context.Context, c *Curriculum) error
	GetByID(ctx context.Context, id uuid.UUID) (*Curriculum, error)
	GetByCode(ctx context.Context, code string) (*Curriculum, error)
	List(ctx context.Context, filter ListCurriculaFilter, p pagination.Params) ([]Curriculum, int, error)
	Update(ctx context.Context, c *Curriculum) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	GetStats(ctx context.Context, curriculumID uuid.UUID) (*CurriculumStats, error)
	CheckStudyProgramActive(ctx context.Context, studyProgramID uuid.UUID) (bool, error)

	AddCourse(ctx context.Context, cc *CurriculumCourse) error
	GetCourse(ctx context.Context, curriculumID, courseID uuid.UUID) (*CurriculumCourseDetail, error)
	ListCourses(ctx context.Context, curriculumID uuid.UUID, filter ListCurriculumCoursesFilter, p pagination.Params) ([]CurriculumCourseDetail, int, error)
	UpdateCourse(ctx context.Context, curriculumID, courseID uuid.UUID, semester *int, category *string) error
	RemoveCourse(ctx context.Context, curriculumID, courseID uuid.UUID) error
	CheckCourseActive(ctx context.Context, courseID uuid.UUID) (bool, error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL repository for curricula.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) Create(ctx context.Context, c *Curriculum) error {
	query := `
		INSERT INTO curricula (study_program_id, code, name, year, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		c.StudyProgramID, c.Code, c.Name, c.Year, c.Description, c.Status, c.CreatedAt, c.UpdatedAt,
	).Scan(&c.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return ErrDuplicateCode
			}
			if pgErr.Code == pgerrcode.ForeignKeyViolation {
				return ErrStudyProgramNotFound
			}
		}
		return fmt.Errorf("failed to insert curriculum: %w", err)
	}
	return nil
}

func (r *pgRepository) GetByID(ctx context.Context, id uuid.UUID) (*Curriculum, error) {
	query := `
		SELECT id, study_program_id, code, name, year, description, status, created_at, updated_at
		FROM curricula
		WHERE id = $1
	`
	var c Curriculum
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.StudyProgramID, &c.Code, &c.Name, &c.Year, &c.Description, &c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get curriculum by id: %w", err)
	}
	return &c, nil
}

func (r *pgRepository) GetByCode(ctx context.Context, code string) (*Curriculum, error) {
	query := `
		SELECT id, study_program_id, code, name, year, description, status, created_at, updated_at
		FROM curricula
		WHERE UPPER(code) = UPPER($1)
	`
	var c Curriculum
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&c.ID, &c.StudyProgramID, &c.Code, &c.Name, &c.Year, &c.Description, &c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get curriculum by code: %w", err)
	}
	return &c, nil
}

func (r *pgRepository) List(ctx context.Context, filter ListCurriculaFilter, p pagination.Params) ([]Curriculum, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(code ILIKE $%d OR name ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.StudyProgramID != nil {
		conditions = append(conditions, fmt.Sprintf("study_program_id = $%d", argIdx))
		args = append(args, *filter.StudyProgramID)
		argIdx++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, strings.ToUpper(filter.Status))
		argIdx++
	}

	if filter.Year != nil {
		conditions = append(conditions, fmt.Sprintf("year = $%d", argIdx))
		args = append(args, *filter.Year)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM curricula %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count curricula: %w", err)
	}

	sortCol := "created_at"
	switch p.Sort {
	case "code":
		sortCol = "code"
	case "name":
		sortCol = "name"
	case "year":
		sortCol = "year"
	}

	sortOrder := "DESC"
	if strings.ToUpper(p.Order) == "ASC" {
		sortOrder = "ASC"
	}

	selectQuery := fmt.Sprintf(`
		SELECT id, study_program_id, code, name, year, description, status, created_at, updated_at
		FROM curricula
		%s
		ORDER BY %s %s, id %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortCol, sortOrder, sortOrder, argIdx, argIdx+1)

	args = append(args, p.Limit, p.Offset())

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list curricula: %w", err)
	}
	defer rows.Close()

	var list []Curriculum
	for rows.Next() {
		var c Curriculum
		if err := rows.Scan(
			&c.ID, &c.StudyProgramID, &c.Code, &c.Name, &c.Year, &c.Description, &c.Status, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan curriculum: %w", err)
		}
		list = append(list, c)
	}

	return list, total, nil
}

func (r *pgRepository) Update(ctx context.Context, c *Curriculum) error {
	query := `
		UPDATE curricula
		SET name = $1, year = $2, description = $3, updated_at = $4
		WHERE id = $5
	`
	cmd, err := r.pool.Exec(ctx, query, c.Name, c.Year, c.Description, c.UpdatedAt, c.ID)
	if err != nil {
		return fmt.Errorf("failed to update curriculum: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE curricula
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	cmd, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update curriculum status: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgRepository) GetStats(ctx context.Context, curriculumID uuid.UUID) (*CurriculumStats, error) {
	query := `
		SELECT 
			COUNT(cc.id) AS total_courses,
			COALESCE(SUM(c.credits), 0) AS total_credits,
			COALESCE(SUM(CASE WHEN cc.category = 'REQUIRED' THEN c.credits ELSE 0 END), 0) AS required_credits,
			COALESCE(SUM(CASE WHEN cc.category = 'ELECTIVE' THEN c.credits ELSE 0 END), 0) AS elective_credits
		FROM curriculum_courses cc
		JOIN courses c ON cc.course_id = c.id
		WHERE cc.curriculum_id = $1
	`
	var stats CurriculumStats
	err := r.pool.QueryRow(ctx, query, curriculumID).Scan(
		&stats.TotalCourses, &stats.TotalCredits, &stats.RequiredCredits, &stats.ElectiveCredits,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get curriculum stats: %w", err)
	}
	return &stats, nil
}

func (r *pgRepository) CheckStudyProgramActive(ctx context.Context, studyProgramID uuid.UUID) (bool, error) {
	var isActive bool
	query := `SELECT is_active FROM study_programs WHERE id = $1`
	err := r.pool.QueryRow(ctx, query, studyProgramID).Scan(&isActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrStudyProgramNotFound
		}
		return false, fmt.Errorf("failed to check study program active: %w", err)
	}
	return isActive, nil
}

func (r *pgRepository) AddCourse(ctx context.Context, cc *CurriculumCourse) error {
	query := `
		INSERT INTO curriculum_courses (curriculum_id, course_id, semester, category, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		cc.CurriculumID, cc.CourseID, cc.Semester, cc.Category, cc.CreatedAt, cc.UpdatedAt,
	).Scan(&cc.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return ErrDuplicateCourse
			}
			if pgErr.Code == pgerrcode.ForeignKeyViolation {
				return ErrCourseNotFound
			}
		}
		return fmt.Errorf("failed to add course to curriculum: %w", err)
	}
	return nil
}

func (r *pgRepository) GetCourse(ctx context.Context, curriculumID, courseID uuid.UUID) (*CurriculumCourseDetail, error) {
	query := `
		SELECT cc.id, cc.curriculum_id, cc.course_id, c.code, c.name, c.credits, cc.semester, cc.category
		FROM curriculum_courses cc
		JOIN courses c ON cc.course_id = c.id
		WHERE cc.curriculum_id = $1 AND cc.course_id = $2
	`
	var detail CurriculumCourseDetail
	err := r.pool.QueryRow(ctx, query, curriculumID, courseID).Scan(
		&detail.ID, &detail.CurriculumID, &detail.CourseID,
		&detail.CourseCode, &detail.CourseName, &detail.Credits,
		&detail.Semester, &detail.Category,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCourseNotFound
		}
		return nil, fmt.Errorf("failed to get course in curriculum: %w", err)
	}
	return &detail, nil
}

func (r *pgRepository) ListCourses(ctx context.Context, curriculumID uuid.UUID, filter ListCurriculumCoursesFilter, p pagination.Params) ([]CurriculumCourseDetail, int, error) {
	conditions := []string{"cc.curriculum_id = $1"}
	args := []any{curriculumID}
	argIdx := 2

	if filter.Semester != nil {
		conditions = append(conditions, fmt.Sprintf("cc.semester = $%d", argIdx))
		args = append(args, *filter.Semester)
		argIdx++
	}

	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf("cc.category = $%d", argIdx))
		args = append(args, strings.ToUpper(filter.Category))
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM curriculum_courses cc %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count curriculum courses: %w", err)
	}

	sortCol := "cc.semester"
	switch p.Sort {
	case "code":
		sortCol = "c.code"
	case "name":
		sortCol = "c.name"
	case "credits":
		sortCol = "c.credits"
	}

	sortOrder := "ASC"
	if strings.ToUpper(p.Order) == "DESC" {
		sortOrder = "DESC"
	}

	selectQuery := fmt.Sprintf(`
		SELECT cc.id, cc.curriculum_id, cc.course_id, c.code, c.name, c.credits, cc.semester, cc.category
		FROM curriculum_courses cc
		JOIN courses c ON cc.course_id = c.id
		%s
		ORDER BY %s %s, c.code ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, sortCol, sortOrder, argIdx, argIdx+1)

	args = append(args, p.Limit, p.Offset())

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list curriculum courses: %w", err)
	}
	defer rows.Close()

	var list []CurriculumCourseDetail
	for rows.Next() {
		var d CurriculumCourseDetail
		if err := rows.Scan(
			&d.ID, &d.CurriculumID, &d.CourseID,
			&d.CourseCode, &d.CourseName, &d.Credits,
			&d.Semester, &d.Category,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan curriculum course detail: %w", err)
		}
		list = append(list, d)
	}

	return list, total, nil
}

func (r *pgRepository) UpdateCourse(ctx context.Context, curriculumID, courseID uuid.UUID, semester *int, category *string) error {
	var sets []string
	var args []any
	argIdx := 1

	if semester != nil {
		sets = append(sets, fmt.Sprintf("semester = $%d", argIdx))
		args = append(args, *semester)
		argIdx++
	}

	if category != nil {
		sets = append(sets, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, strings.ToUpper(*category))
		argIdx++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	query := fmt.Sprintf(`
		UPDATE curriculum_courses
		SET %s
		WHERE curriculum_id = $%d AND course_id = $%d
	`, strings.Join(sets, ", "), argIdx, argIdx+1)

	args = append(args, curriculumID, courseID)

	cmd, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update curriculum course: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrCourseNotFound
	}
	return nil
}

func (r *pgRepository) RemoveCourse(ctx context.Context, curriculumID, courseID uuid.UUID) error {
	query := `
		DELETE FROM curriculum_courses
		WHERE curriculum_id = $1 AND course_id = $2
	`
	cmd, err := r.pool.Exec(ctx, query, curriculumID, courseID)
	if err != nil {
		return fmt.Errorf("failed to remove course from curriculum: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrCourseNotFound
	}
	return nil
}

func (r *pgRepository) CheckCourseActive(ctx context.Context, courseID uuid.UUID) (bool, error) {
	var isActive bool
	query := `SELECT is_active FROM courses WHERE id = $1`
	err := r.pool.QueryRow(ctx, query, courseID).Scan(&isActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrCourseNotFound
		}
		return false, fmt.Errorf("failed to check course active: %w", err)
	}
	return isActive, nil
}
