package lecturers

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
	ErrNotFound             = errors.New("lecturer not found")
	ErrDuplicateNIDN        = errors.New("NIDN already registered")
	ErrDuplicateEmail       = errors.New("lecturer email already registered")
	ErrStudyProgramNotFound = errors.New("study program not found")
	ErrUserNotFound         = errors.New("linked user account not found")
	ErrUserAlreadyLinked    = errors.New("user account is already linked to another lecturer")
)

// Repository defines the persistence interface for lecturers.
type Repository interface {
	Create(ctx context.Context, l *Lecturer) error
	GetByID(ctx context.Context, id uuid.UUID) (*Lecturer, error)
	GetByNIDN(ctx context.Context, nidn string) (*Lecturer, error)
	GetByEmail(ctx context.Context, email string) (*Lecturer, error)
	List(ctx context.Context, filter LecturerFilter, p pagination.Params) ([]Lecturer, int, error)
	Update(ctx context.Context, l *Lecturer) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	CheckStudyProgramActive(ctx context.Context, spID uuid.UUID) (exists bool, active bool, err error)
	CheckUserExists(ctx context.Context, userID uuid.UUID) (bool, error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL implementation of Repository.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) CheckStudyProgramActive(ctx context.Context, spID uuid.UUID) (bool, bool, error) {
	query := `SELECT is_active FROM study_programs WHERE id = $1`
	var isActive bool
	err := r.pool.QueryRow(ctx, query, spID).Scan(&isActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, nil
		}
		return false, false, fmt.Errorf("failed to check study program: %w", err)
	}
	return true, isActive, nil
}

func (r *pgRepository) CheckUserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user: %w", err)
	}
	return exists, nil
}

func (r *pgRepository) Create(ctx context.Context, l *Lecturer) error {
	query := `
		INSERT INTO lecturers (user_id, study_program_id, nidn, nip, name, title_prefix, title_suffix, email, phone, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		l.UserID, l.StudyProgramID, l.NIDN, l.NIP, l.Name, l.TitlePrefix, l.TitleSuffix,
		l.Email, l.Phone, l.Status, l.CreatedAt, l.UpdatedAt,
	).Scan(&l.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				if strings.Contains(pgErr.ConstraintName, "nidn") {
					return ErrDuplicateNIDN
				}
				if strings.Contains(pgErr.ConstraintName, "email") {
					return ErrDuplicateEmail
				}
				if strings.Contains(pgErr.ConstraintName, "user_id") {
					return ErrUserAlreadyLinked
				}
			}
			if pgErr.Code == pgerrcode.ForeignKeyViolation {
				if strings.Contains(pgErr.ConstraintName, "study_program") {
					return ErrStudyProgramNotFound
				}
				if strings.Contains(pgErr.ConstraintName, "user") {
					return ErrUserNotFound
				}
			}
		}
		return fmt.Errorf("failed to insert lecturer: %w", err)
	}
	return nil
}

func (r *pgRepository) GetByID(ctx context.Context, id uuid.UUID) (*Lecturer, error) {
	query := `
		SELECT l.id, l.user_id, l.study_program_id, l.nidn, l.nip, l.name, l.title_prefix, l.title_suffix,
		       l.email, l.phone, l.status, l.created_at, l.updated_at,
		       sp.code as study_program_code, sp.name as study_program_name, f.name as faculty_name
		FROM lecturers l
		JOIN study_programs sp ON sp.id = l.study_program_id
		JOIN faculties f ON f.id = sp.faculty_id
		WHERE l.id = $1
	`
	var l Lecturer
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&l.ID, &l.UserID, &l.StudyProgramID, &l.NIDN, &l.NIP, &l.Name, &l.TitlePrefix, &l.TitleSuffix,
		&l.Email, &l.Phone, &l.Status, &l.CreatedAt, &l.UpdatedAt,
		&l.StudyProgramCode, &l.StudyProgramName, &l.FacultyName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query lecturer by id: %w", err)
	}
	return &l, nil
}

func (r *pgRepository) GetByNIDN(ctx context.Context, nidn string) (*Lecturer, error) {
	query := `
		SELECT l.id, l.user_id, l.study_program_id, l.nidn, l.nip, l.name, l.title_prefix, l.title_suffix,
		       l.email, l.phone, l.status, l.created_at, l.updated_at,
		       sp.code as study_program_code, sp.name as study_program_name, f.name as faculty_name
		FROM lecturers l
		JOIN study_programs sp ON sp.id = l.study_program_id
		JOIN faculties f ON f.id = sp.faculty_id
		WHERE l.nidn = $1
	`
	var l Lecturer
	err := r.pool.QueryRow(ctx, query, nidn).Scan(
		&l.ID, &l.UserID, &l.StudyProgramID, &l.NIDN, &l.NIP, &l.Name, &l.TitlePrefix, &l.TitleSuffix,
		&l.Email, &l.Phone, &l.Status, &l.CreatedAt, &l.UpdatedAt,
		&l.StudyProgramCode, &l.StudyProgramName, &l.FacultyName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query lecturer by nidn: %w", err)
	}
	return &l, nil
}

func (r *pgRepository) GetByEmail(ctx context.Context, email string) (*Lecturer, error) {
	query := `
		SELECT l.id, l.user_id, l.study_program_id, l.nidn, l.nip, l.name, l.title_prefix, l.title_suffix,
		       l.email, l.phone, l.status, l.created_at, l.updated_at,
		       sp.code as study_program_code, sp.name as study_program_name, f.name as faculty_name
		FROM lecturers l
		JOIN study_programs sp ON sp.id = l.study_program_id
		JOIN faculties f ON f.id = sp.faculty_id
		WHERE l.email = $1
	`
	var l Lecturer
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&l.ID, &l.UserID, &l.StudyProgramID, &l.NIDN, &l.NIP, &l.Name, &l.TitlePrefix, &l.TitleSuffix,
		&l.Email, &l.Phone, &l.Status, &l.CreatedAt, &l.UpdatedAt,
		&l.StudyProgramCode, &l.StudyProgramName, &l.FacultyName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query lecturer by email: %w", err)
	}
	return &l, nil
}

func (r *pgRepository) List(ctx context.Context, filter LecturerFilter, p pagination.Params) ([]Lecturer, int, error) {
	whereClauses := []string{"1=1"}
	args := []any{}
	argIdx := 1

	if filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(l.nidn ILIKE $%d OR l.name ILIKE $%d OR l.email ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.StudyProgramID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("l.study_program_id = $%d", argIdx))
		args = append(args, *filter.StudyProgramID)
		argIdx++
	}

	if filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("l.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM lecturers l WHERE %s", whereSQL)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count lecturers: %w", err)
	}

	// Sorting
	sortCol := "l.name"
	if p.Sort != "" {
		switch strings.ToLower(p.Sort) {
		case "nidn":
			sortCol = "l.nidn"
		case "created_at":
			sortCol = "l.created_at"
		case "status":
			sortCol = "l.status"
		default:
			sortCol = "l.name"
		}
	}
	sortOrder := "ASC"
	if strings.ToLower(p.Order) == "desc" {
		sortOrder = "DESC"
	}

	query := fmt.Sprintf(`
		SELECT l.id, l.user_id, l.study_program_id, l.nidn, l.nip, l.name, l.title_prefix, l.title_suffix,
		       l.email, l.phone, l.status, l.created_at, l.updated_at,
		       sp.code as study_program_code, sp.name as study_program_name, f.name as faculty_name
		FROM lecturers l
		JOIN study_programs sp ON sp.id = l.study_program_id
		JOIN faculties f ON f.id = sp.faculty_id
		WHERE %s
		ORDER BY %s %s, l.id ASC
		LIMIT $%d OFFSET $%d
	`, whereSQL, sortCol, sortOrder, argIdx, argIdx+1)

	args = append(args, p.Limit, p.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list lecturers: %w", err)
	}
	defer rows.Close()

	items := make([]Lecturer, 0)
	for rows.Next() {
		var l Lecturer
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.StudyProgramID, &l.NIDN, &l.NIP, &l.Name, &l.TitlePrefix, &l.TitleSuffix,
			&l.Email, &l.Phone, &l.Status, &l.CreatedAt, &l.UpdatedAt,
			&l.StudyProgramCode, &l.StudyProgramName, &l.FacultyName,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan lecturer: %w", err)
		}
		items = append(items, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error while listing lecturers: %w", err)
	}

	return items, total, nil
}

func (r *pgRepository) Update(ctx context.Context, l *Lecturer) error {
	query := `
		UPDATE lecturers
		SET user_id = $1, study_program_id = $2, nidn = $3, nip = $4, name = $5,
		    title_prefix = $6, title_suffix = $7, email = $8, phone = $9, updated_at = $10
		WHERE id = $11
	`
	cmdTag, err := r.pool.Exec(ctx, query,
		l.UserID, l.StudyProgramID, l.NIDN, l.NIP, l.Name, l.TitlePrefix, l.TitleSuffix,
		l.Email, l.Phone, l.UpdatedAt, l.ID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				if strings.Contains(pgErr.ConstraintName, "nidn") {
					return ErrDuplicateNIDN
				}
				if strings.Contains(pgErr.ConstraintName, "email") {
					return ErrDuplicateEmail
				}
				if strings.Contains(pgErr.ConstraintName, "user_id") {
					return ErrUserAlreadyLinked
				}
			}
			if pgErr.Code == pgerrcode.ForeignKeyViolation {
				if strings.Contains(pgErr.ConstraintName, "study_program") {
					return ErrStudyProgramNotFound
				}
				if strings.Contains(pgErr.ConstraintName, "user") {
					return ErrUserNotFound
				}
			}
		}
		return fmt.Errorf("failed to update lecturer: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE lecturers
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	cmdTag, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update lecturer status: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
