package faculties

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
	ErrNotFound         = errors.New("faculty not found")
	ErrDuplicateCode    = errors.New("faculty code already exists")
	ErrHasStudyPrograms = errors.New("cannot delete or modify faculty with linked study programs")
)

// Repository defines the interface for faculty persistence operations.
type Repository interface {
	Create(ctx context.Context, f *Faculty) error
	GetByID(ctx context.Context, id uuid.UUID) (*Faculty, error)
	GetByCode(ctx context.Context, code string) (*Faculty, error)
	List(ctx context.Context, filter FacultyFilter, p pagination.Params) ([]Faculty, int, error)
	Update(ctx context.Context, f *Faculty) error
	UpdateStatus(ctx context.Context, id uuid.UUID, isActive bool) error
	CountStudyPrograms(ctx context.Context, id uuid.UUID) (int, error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL implementation of Repository.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) Create(ctx context.Context, f *Faculty) error {
	query := `
		INSERT INTO faculties (code, name, description, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		f.Code, f.Name, f.Description, f.IsActive, f.CreatedAt, f.UpdatedAt,
	).Scan(&f.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return ErrDuplicateCode
		}
		return fmt.Errorf("failed to insert faculty: %w", err)
	}
	return nil
}

func (r *pgRepository) GetByID(ctx context.Context, id uuid.UUID) (*Faculty, error) {
	query := `
		SELECT id, code, name, description, is_active, created_at, updated_at
		FROM faculties
		WHERE id = $1
	`
	var f Faculty
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&f.ID, &f.Code, &f.Name, &f.Description, &f.IsActive, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query faculty by id: %w", err)
	}
	return &f, nil
}

func (r *pgRepository) GetByCode(ctx context.Context, code string) (*Faculty, error) {
	query := `
		SELECT id, code, name, description, is_active, created_at, updated_at
		FROM faculties
		WHERE code = $1
	`
	var f Faculty
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&f.ID, &f.Code, &f.Name, &f.Description, &f.IsActive, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query faculty by code: %w", err)
	}
	return &f, nil
}

func (r *pgRepository) List(ctx context.Context, filter FacultyFilter, p pagination.Params) ([]Faculty, int, error) {
	whereClauses := []string{"1=1"}
	args := []any{}
	argIdx := 1

	if filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(code ILIKE $%d OR name ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.IsActive != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *filter.IsActive)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM faculties WHERE %s", whereSQL)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count faculties: %w", err)
	}

	// Sorting
	sortCol := "name"
	if p.Sort != "" {
		sortCol = strings.ToLower(p.Sort)
	}
	sortOrder := "ASC"
	if strings.ToLower(p.Order) == "desc" {
		sortOrder = "DESC"
	}

	query := fmt.Sprintf(`
		SELECT id, code, name, description, is_active, created_at, updated_at
		FROM faculties
		WHERE %s
		ORDER BY %s %s, id ASC
		LIMIT $%d OFFSET $%d
	`, whereSQL, sortCol, sortOrder, argIdx, argIdx+1)

	args = append(args, p.Limit, p.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list faculties: %w", err)
	}
	defer rows.Close()

	faculties := make([]Faculty, 0)
	for rows.Next() {
		var f Faculty
		if err := rows.Scan(&f.ID, &f.Code, &f.Name, &f.Description, &f.IsActive, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan faculty: %w", err)
		}
		faculties = append(faculties, f)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error while listing faculties: %w", err)
	}

	return faculties, total, nil
}

func (r *pgRepository) Update(ctx context.Context, f *Faculty) error {
	query := `
		UPDATE faculties
		SET code = $1, name = $2, description = $3, updated_at = $4
		WHERE id = $5
	`
	cmdTag, err := r.pool.Exec(ctx, query, f.Code, f.Name, f.Description, f.UpdatedAt, f.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return ErrDuplicateCode
		}
		return fmt.Errorf("failed to update faculty: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgRepository) UpdateStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	query := `
		UPDATE faculties
		SET is_active = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	cmdTag, err := r.pool.Exec(ctx, query, isActive, id)
	if err != nil {
		return fmt.Errorf("failed to update faculty status: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgRepository) CountStudyPrograms(ctx context.Context, id uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM study_programs WHERE faculty_id = $1`
	var count int
	if err := r.pool.QueryRow(ctx, query, id).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count linked study programs: %w", err)
	}
	return count, nil
}
