package courses

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
	ErrNotFound        = errors.New("course not found")
	ErrDuplicateCode   = errors.New("course code already exists")
	ErrDuplicatePrereq = errors.New("prerequisite already exists")
	ErrSelfPrereq      = errors.New("course cannot be prerequisite of itself")
	ErrCourseLinked    = errors.New("cannot delete or modify course linked to curricula")
)

// Repository defines the persistence interface for academic courses.
type Repository interface {
	Create(ctx context.Context, c *Course) error
	GetByID(ctx context.Context, id uuid.UUID) (*Course, error)
	GetByCode(ctx context.Context, code string) (*Course, error)
	List(ctx context.Context, filter ListCoursesFilter, p pagination.Params) ([]Course, int, error)
	Update(ctx context.Context, c *Course) error
	UpdateStatus(ctx context.Context, id uuid.UUID, isActive bool) error
	AddPrerequisite(ctx context.Context, courseID, prereqID uuid.UUID) error
	RemovePrerequisite(ctx context.Context, courseID, prereqID uuid.UUID) error
	GetPrerequisites(ctx context.Context, courseID uuid.UUID) ([]PrerequisiteDetail, error)
	IsPrerequisite(ctx context.Context, courseID, prereqID uuid.UUID) (bool, error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL repository instance for courses.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) Create(ctx context.Context, c *Course) error {
	query := `
		INSERT INTO courses (code, name, description, credits, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		c.Code, c.Name, c.Description, c.Credits, c.IsActive, c.CreatedAt, c.UpdatedAt,
	).Scan(&c.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return ErrDuplicateCode
		}
		return fmt.Errorf("failed to insert course: %w", err)
	}
	return nil
}

func (r *pgRepository) GetByID(ctx context.Context, id uuid.UUID) (*Course, error) {
	query := `
		SELECT id, code, name, description, credits, is_active, created_at, updated_at
		FROM courses
		WHERE id = $1
	`
	var c Course
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Code, &c.Name, &c.Description, &c.Credits, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get course by id: %w", err)
	}
	return &c, nil
}

func (r *pgRepository) GetByCode(ctx context.Context, code string) (*Course, error) {
	query := `
		SELECT id, code, name, description, credits, is_active, created_at, updated_at
		FROM courses
		WHERE UPPER(code) = UPPER($1)
	`
	var c Course
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&c.ID, &c.Code, &c.Name, &c.Description, &c.Credits, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get course by code: %w", err)
	}
	return &c, nil
}

func (r *pgRepository) List(ctx context.Context, filter ListCoursesFilter, p pagination.Params) ([]Course, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(code ILIKE $%d OR name ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.Credits != nil {
		conditions = append(conditions, fmt.Sprintf("credits = $%d", argIdx))
		args = append(args, *filter.Credits)
		argIdx++
	}

	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *filter.IsActive)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM courses %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count courses: %w", err)
	}

	sortCol := "created_at"
	switch p.Sort {
	case "code":
		sortCol = "code"
	case "name":
		sortCol = "name"
	case "credits":
		sortCol = "credits"
	}

	sortOrder := "DESC"
	if strings.ToUpper(p.Order) == "ASC" {
		sortOrder = "ASC"
	}

	selectQuery := fmt.Sprintf(`
		SELECT id, code, name, description, credits, is_active, created_at, updated_at
		FROM courses
		%s
		ORDER BY %s %s, id %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortCol, sortOrder, sortOrder, argIdx, argIdx+1)

	args = append(args, p.Limit, p.Offset())

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list courses: %w", err)
	}
	defer rows.Close()

	var list []Course
	for rows.Next() {
		var c Course
		if err := rows.Scan(
			&c.ID, &c.Code, &c.Name, &c.Description, &c.Credits, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan course: %w", err)
		}
		list = append(list, c)
	}

	return list, total, nil
}

func (r *pgRepository) Update(ctx context.Context, c *Course) error {
	query := `
		UPDATE courses
		SET name = $1, description = $2, credits = $3, updated_at = $4
		WHERE id = $5
	`
	cmd, err := r.pool.Exec(ctx, query, c.Name, c.Description, c.Credits, c.UpdatedAt, c.ID)
	if err != nil {
		return fmt.Errorf("failed to update course: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgRepository) UpdateStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	query := `
		UPDATE courses
		SET is_active = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	cmd, err := r.pool.Exec(ctx, query, isActive, id)
	if err != nil {
		return fmt.Errorf("failed to update course status: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgRepository) AddPrerequisite(ctx context.Context, courseID, prereqID uuid.UUID) error {
	query := `
		INSERT INTO course_prerequisites (course_id, prerequisite_course_id)
		VALUES ($1, $2)
	`
	_, err := r.pool.Exec(ctx, query, courseID, prereqID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return ErrDuplicatePrereq
			}
			if pgErr.Code == pgerrcode.CheckViolation {
				return ErrSelfPrereq
			}
			if pgErr.Code == pgerrcode.ForeignKeyViolation {
				return ErrNotFound
			}
		}
		return fmt.Errorf("failed to add prerequisite: %w", err)
	}
	return nil
}

func (r *pgRepository) RemovePrerequisite(ctx context.Context, courseID, prereqID uuid.UUID) error {
	query := `
		DELETE FROM course_prerequisites
		WHERE course_id = $1 AND prerequisite_course_id = $2
	`
	cmd, err := r.pool.Exec(ctx, query, courseID, prereqID)
	if err != nil {
		return fmt.Errorf("failed to remove prerequisite: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgRepository) GetPrerequisites(ctx context.Context, courseID uuid.UUID) ([]PrerequisiteDetail, error) {
	query := `
		SELECT c.id, c.code, c.name, c.credits
		FROM course_prerequisites cp
		JOIN courses c ON cp.prerequisite_course_id = c.id
		WHERE cp.course_id = $1
		ORDER BY c.code ASC
	`
	rows, err := r.pool.Query(ctx, query, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get prerequisites: %w", err)
	}
	defer rows.Close()

	var prereqs []PrerequisiteDetail
	for rows.Next() {
		var p PrerequisiteDetail
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.Credits); err != nil {
			return nil, fmt.Errorf("failed to scan prerequisite: %w", err)
		}
		prereqs = append(prereqs, p)
	}
	return prereqs, nil
}

func (r *pgRepository) IsPrerequisite(ctx context.Context, courseID, prereqID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM course_prerequisites
			WHERE course_id = $1 AND prerequisite_course_id = $2
		)
	`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, courseID, prereqID).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check prerequisite: %w", err)
	}
	return exists, nil
}
