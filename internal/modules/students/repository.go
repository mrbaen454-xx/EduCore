package students

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
	ErrNotFound             = errors.New("student not found")
	ErrDuplicateNIM         = errors.New("NIM already registered")
	ErrDuplicateEmail       = errors.New("student email already registered")
	ErrStudyProgramNotFound = errors.New("study program not found")
	ErrUserNotFound         = errors.New("linked user account not found")
	ErrUserAlreadyLinked    = errors.New("user account is already linked to another student")
)

// Repository defines the persistence interface for students.
type Repository interface {
	Create(ctx context.Context, s *Student) error
	GetByID(ctx context.Context, id uuid.UUID) (*Student, error)
	GetByNIM(ctx context.Context, nim string) (*Student, error)
	GetByEmail(ctx context.Context, email string) (*Student, error)
	List(ctx context.Context, filter StudentFilter, p pagination.Params) ([]Student, int, error)
	Update(ctx context.Context, s *Student) error
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

func (r *pgRepository) Create(ctx context.Context, s *Student) error {
	query := `
		INSERT INTO students (user_id, study_program_id, nim, name, email, phone, gender, birth_date, admission_year, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		s.UserID, s.StudyProgramID, s.NIM, s.Name, s.Email, s.Phone, s.Gender, s.BirthDate,
		s.AdmissionYear, s.Status, s.CreatedAt, s.UpdatedAt,
	).Scan(&s.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				if strings.Contains(pgErr.ConstraintName, "nim") {
					return ErrDuplicateNIM
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
		return fmt.Errorf("failed to insert student: %w", err)
	}
	return nil
}

func (r *pgRepository) GetByID(ctx context.Context, id uuid.UUID) (*Student, error) {
	query := `
		SELECT s.id, s.user_id, s.study_program_id, s.nim, s.name, s.email, s.phone, s.gender,
		       s.birth_date, s.admission_year, s.status, s.created_at, s.updated_at,
		       sp.code as study_program_code, sp.name as study_program_name, f.name as faculty_name
		FROM students s
		JOIN study_programs sp ON sp.id = s.study_program_id
		JOIN faculties f ON f.id = sp.faculty_id
		WHERE s.id = $1
	`
	var s Student
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.UserID, &s.StudyProgramID, &s.NIM, &s.Name, &s.Email, &s.Phone, &s.Gender,
		&s.BirthDate, &s.AdmissionYear, &s.Status, &s.CreatedAt, &s.UpdatedAt,
		&s.StudyProgramCode, &s.StudyProgramName, &s.FacultyName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query student by id: %w", err)
	}
	return &s, nil
}

func (r *pgRepository) GetByNIM(ctx context.Context, nim string) (*Student, error) {
	query := `
		SELECT s.id, s.user_id, s.study_program_id, s.nim, s.name, s.email, s.phone, s.gender,
		       s.birth_date, s.admission_year, s.status, s.created_at, s.updated_at,
		       sp.code as study_program_code, sp.name as study_program_name, f.name as faculty_name
		FROM students s
		JOIN study_programs sp ON sp.id = s.study_program_id
		JOIN faculties f ON f.id = sp.faculty_id
		WHERE s.nim = $1
	`
	var s Student
	err := r.pool.QueryRow(ctx, query, nim).Scan(
		&s.ID, &s.UserID, &s.StudyProgramID, &s.NIM, &s.Name, &s.Email, &s.Phone, &s.Gender,
		&s.BirthDate, &s.AdmissionYear, &s.Status, &s.CreatedAt, &s.UpdatedAt,
		&s.StudyProgramCode, &s.StudyProgramName, &s.FacultyName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query student by nim: %w", err)
	}
	return &s, nil
}

func (r *pgRepository) GetByEmail(ctx context.Context, email string) (*Student, error) {
	query := `
		SELECT s.id, s.user_id, s.study_program_id, s.nim, s.name, s.email, s.phone, s.gender,
		       s.birth_date, s.admission_year, s.status, s.created_at, s.updated_at,
		       sp.code as study_program_code, sp.name as study_program_name, f.name as faculty_name
		FROM students s
		JOIN study_programs sp ON sp.id = s.study_program_id
		JOIN faculties f ON f.id = sp.faculty_id
		WHERE s.email = $1
	`
	var s Student
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&s.ID, &s.UserID, &s.StudyProgramID, &s.NIM, &s.Name, &s.Email, &s.Phone, &s.Gender,
		&s.BirthDate, &s.AdmissionYear, &s.Status, &s.CreatedAt, &s.UpdatedAt,
		&s.StudyProgramCode, &s.StudyProgramName, &s.FacultyName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query student by email: %w", err)
	}
	return &s, nil
}

func (r *pgRepository) List(ctx context.Context, filter StudentFilter, p pagination.Params) ([]Student, int, error) {
	whereClauses := []string{"1=1"}
	args := []any{}
	argIdx := 1

	if filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(s.nim ILIKE $%d OR s.name ILIKE $%d OR s.email ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.StudyProgramID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("s.study_program_id = $%d", argIdx))
		args = append(args, *filter.StudyProgramID)
		argIdx++
	}

	if filter.AdmissionYear != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("s.admission_year = $%d", argIdx))
		args = append(args, *filter.AdmissionYear)
		argIdx++
	}

	if filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("s.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM students s WHERE %s", whereSQL)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count students: %w", err)
	}

	// Sorting
	sortCol := "s.name"
	if p.Sort != "" {
		switch strings.ToLower(p.Sort) {
		case "nim":
			sortCol = "s.nim"
		case "admission_year":
			sortCol = "s.admission_year"
		case "created_at":
			sortCol = "s.created_at"
		case "status":
			sortCol = "s.status"
		default:
			sortCol = "s.name"
		}
	}
	sortOrder := "ASC"
	if strings.ToLower(p.Order) == "desc" {
		sortOrder = "DESC"
	}

	query := fmt.Sprintf(`
		SELECT s.id, s.user_id, s.study_program_id, s.nim, s.name, s.email, s.phone, s.gender,
		       s.birth_date, s.admission_year, s.status, s.created_at, s.updated_at,
		       sp.code as study_program_code, sp.name as study_program_name, f.name as faculty_name
		FROM students s
		JOIN study_programs sp ON sp.id = s.study_program_id
		JOIN faculties f ON f.id = sp.faculty_id
		WHERE %s
		ORDER BY %s %s, s.id ASC
		LIMIT $%d OFFSET $%d
	`, whereSQL, sortCol, sortOrder, argIdx, argIdx+1)

	args = append(args, p.Limit, p.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list students: %w", err)
	}
	defer rows.Close()

	items := make([]Student, 0)
	for rows.Next() {
		var s Student
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.StudyProgramID, &s.NIM, &s.Name, &s.Email, &s.Phone, &s.Gender,
			&s.BirthDate, &s.AdmissionYear, &s.Status, &s.CreatedAt, &s.UpdatedAt,
			&s.StudyProgramCode, &s.StudyProgramName, &s.FacultyName,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan student: %w", err)
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error while listing students: %w", err)
	}

	return items, total, nil
}

func (r *pgRepository) Update(ctx context.Context, s *Student) error {
	query := `
		UPDATE students
		SET user_id = $1, study_program_id = $2, nim = $3, name = $4, email = $5,
		    phone = $6, gender = $7, birth_date = $8, admission_year = $9, updated_at = $10
		WHERE id = $11
	`
	cmdTag, err := r.pool.Exec(ctx, query,
		s.UserID, s.StudyProgramID, s.NIM, s.Name, s.Email, s.Phone, s.Gender,
		s.BirthDate, s.AdmissionYear, s.UpdatedAt, s.ID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				if strings.Contains(pgErr.ConstraintName, "nim") {
					return ErrDuplicateNIM
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
		return fmt.Errorf("failed to update student: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE students
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	cmdTag, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update student status: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
