package study_programs

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
	ErrNotFound        = errors.New("study program not found")
	ErrDuplicateCode   = errors.New("study program code already exists")
	ErrFacultyNotFound = errors.New("parent faculty not found")
	ErrFacultyInactive = errors.New("parent faculty is inactive")
)

// Repository defines the persistence interface for study programs.
type Repository interface {
	Create(ctx context.Context, sp *StudyProgram) error
	GetByID(ctx context.Context, id uuid.UUID) (*StudyProgram, error)
	GetByCode(ctx context.Context, code string) (*StudyProgram, error)
	List(ctx context.Context, filter StudyProgramFilter, p pagination.Params) ([]StudyProgram, int, error)
	Update(ctx context.Context, sp *StudyProgram) error
	UpdateStatus(ctx context.Context, id uuid.UUID, isActive bool) error
	CheckFacultyActive(ctx context.Context, facultyID uuid.UUID) (exists bool, active bool, err error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL implementation of Repository.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) CheckFacultyActive(ctx context.Context, facultyID uuid.UUID) (bool, bool, error) {
	query := `SELECT is_active FROM faculties WHERE id = $1`
	var isActive bool
	err := r.pool.QueryRow(ctx, query, facultyID).Scan(&isActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, nil
		}
		return false, false, fmt.Errorf("failed to check faculty: %w", err)
	}
	return true, isActive, nil
}

func (r *pgRepository) Create(ctx context.Context, sp *StudyProgram) error {
	query := `
		INSERT INTO study_programs (faculty_id, code, name, level, description, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		sp.FacultyID, sp.Code, sp.Name, sp.Level, sp.Description, sp.IsActive, sp.CreatedAt, sp.UpdatedAt,
	).Scan(&sp.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return ErrDuplicateCode
			}
			if pgErr.Code == pgerrcode.ForeignKeyViolation {
				return ErrFacultyNotFound
			}
		}
		return fmt.Errorf("failed to insert study program: %w", err)
	}
	return nil
}

func (r *pgRepository) GetByID(ctx context.Context, id uuid.UUID) (*StudyProgram, error) {
	query := `
		SELECT sp.id, sp.faculty_id, sp.code, sp.name, sp.level, sp.description, sp.is_active, 
		       sp.created_at, sp.updated_at, f.code as faculty_code, f.name as faculty_name
		FROM study_programs sp
		JOIN faculties f ON f.id = sp.faculty_id
		WHERE sp.id = $1
	`
	var sp StudyProgram
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&sp.ID, &sp.FacultyID, &sp.Code, &sp.Name, &sp.Level, &sp.Description, &sp.IsActive,
		&sp.CreatedAt, &sp.UpdatedAt, &sp.FacultyCode, &sp.FacultyName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query study program by id: %w", err)
	}
	return &sp, nil
}

func (r *pgRepository) GetByCode(ctx context.Context, code string) (*StudyProgram, error) {
	query := `
		SELECT sp.id, sp.faculty_id, sp.code, sp.name, sp.level, sp.description, sp.is_active, 
		       sp.created_at, sp.updated_at, f.code as faculty_code, f.name as faculty_name
		FROM study_programs sp
		JOIN faculties f ON f.id = sp.faculty_id
		WHERE sp.code = $1
	`
	var sp StudyProgram
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&sp.ID, &sp.FacultyID, &sp.Code, &sp.Name, &sp.Level, &sp.Description, &sp.IsActive,
		&sp.CreatedAt, &sp.UpdatedAt, &sp.FacultyCode, &sp.FacultyName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query study program by code: %w", err)
	}
	return &sp, nil
}

func (r *pgRepository) List(ctx context.Context, filter StudyProgramFilter, p pagination.Params) ([]StudyProgram, int, error) {
	whereClauses := []string{"1=1"}
	args := []any{}
	argIdx := 1

	if filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(sp.code ILIKE $%d OR sp.name ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.FacultyID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("sp.faculty_id = $%d", argIdx))
		args = append(args, *filter.FacultyID)
		argIdx++
	}

	if filter.Level != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("sp.level = $%d", argIdx))
		args = append(args, filter.Level)
		argIdx++
	}

	if filter.IsActive != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("sp.is_active = $%d", argIdx))
		args = append(args, *filter.IsActive)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM study_programs sp WHERE %s", whereSQL)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count study programs: %w", err)
	}

	// Sorting
	sortCol := "sp.name"
	if p.Sort != "" {
		switch strings.ToLower(p.Sort) {
		case "code":
			sortCol = "sp.code"
		case "level":
			sortCol = "sp.level"
		case "created_at":
			sortCol = "sp.created_at"
		case "is_active":
			sortCol = "sp.is_active"
		default:
			sortCol = "sp.name"
		}
	}
	sortOrder := "ASC"
	if strings.ToLower(p.Order) == "desc" {
		sortOrder = "DESC"
	}

	query := fmt.Sprintf(`
		SELECT sp.id, sp.faculty_id, sp.code, sp.name, sp.level, sp.description, sp.is_active, 
		       sp.created_at, sp.updated_at, f.code as faculty_code, f.name as faculty_name
		FROM study_programs sp
		JOIN faculties f ON f.id = sp.faculty_id
		WHERE %s
		ORDER BY %s %s, sp.id ASC
		LIMIT $%d OFFSET $%d
	`, whereSQL, sortCol, sortOrder, argIdx, argIdx+1)

	args = append(args, p.Limit, p.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list study programs: %w", err)
	}
	defer rows.Close()

	items := make([]StudyProgram, 0)
	for rows.Next() {
		var sp StudyProgram
		if err := rows.Scan(
			&sp.ID, &sp.FacultyID, &sp.Code, &sp.Name, &sp.Level, &sp.Description, &sp.IsActive,
			&sp.CreatedAt, &sp.UpdatedAt, &sp.FacultyCode, &sp.FacultyName,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan study program: %w", err)
		}
		items = append(items, sp)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error while listing study programs: %w", err)
	}

	return items, total, nil
}

func (r *pgRepository) Update(ctx context.Context, sp *StudyProgram) error {
	query := `
		UPDATE study_programs
		SET faculty_id = $1, code = $2, name = $3, level = $4, description = $5, updated_at = $6
		WHERE id = $7
	`
	cmdTag, err := r.pool.Exec(ctx, query, sp.FacultyID, sp.Code, sp.Name, sp.Level, sp.Description, sp.UpdatedAt, sp.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return ErrDuplicateCode
			}
			if pgErr.Code == pgerrcode.ForeignKeyViolation {
				return ErrFacultyNotFound
			}
		}
		return fmt.Errorf("failed to update study program: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgRepository) UpdateStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	query := `
		UPDATE study_programs
		SET is_active = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	cmdTag, err := r.pool.Exec(ctx, query, isActive, id)
	if err != nil {
		return fmt.Errorf("failed to update study program status: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
