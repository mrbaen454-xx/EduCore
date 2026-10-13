package academic_periods

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
	ErrNotFound      = appErrors.ErrNotFound("Periode akademik tidak ditemukan")
	ErrDuplicateCode = appErrors.ErrConflict("Kode periode akademik sudah terdaftar")
)

type Repository interface {
	Create(ctx context.Context, p *AcademicPeriod) error
	GetByID(ctx context.Context, id uuid.UUID) (*AcademicPeriod, error)
	GetByCode(ctx context.Context, code string) (*AcademicPeriod, error)
	List(ctx context.Context, filter AcademicPeriodFilter, p pagination.Params) ([]AcademicPeriod, int, error)
	Update(ctx context.Context, p *AcademicPeriod) error
}

type pgRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) Create(ctx context.Context, p *AcademicPeriod) error {
	query := `
		INSERT INTO academic_periods (id, code, name, semester_type, start_date, end_date, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING created_at, updated_at;
	`
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		p.ID, p.Code, p.Name, p.SemesterType, p.StartDate, p.EndDate, p.IsActive,
	).Scan(&p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateCode
		}
		return appErrors.ErrInternal(err)
	}

	return nil
}

func (r *pgRepository) GetByID(ctx context.Context, id uuid.UUID) (*AcademicPeriod, error) {
	query := `
		SELECT id, code, name, semester_type, start_date, end_date, is_active, created_at, updated_at
		FROM academic_periods
		WHERE id = $1;
	`
	var p AcademicPeriod
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Code, &p.Name, &p.SemesterType, &p.StartDate, &p.EndDate, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, appErrors.ErrInternal(err)
	}
	return &p, nil
}

func (r *pgRepository) GetByCode(ctx context.Context, code string) (*AcademicPeriod, error) {
	query := `
		SELECT id, code, name, semester_type, start_date, end_date, is_active, created_at, updated_at
		FROM academic_periods
		WHERE code = $1;
	`
	var p AcademicPeriod
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&p.ID, &p.Code, &p.Name, &p.SemesterType, &p.StartDate, &p.EndDate, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, appErrors.ErrInternal(err)
	}
	return &p, nil
}

func (r *pgRepository) List(ctx context.Context, filter AcademicPeriodFilter, p pagination.Params) ([]AcademicPeriod, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(code ILIKE $%d OR name ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}
	if filter.SemesterType != "" {
		conditions = append(conditions, fmt.Sprintf("semester_type = $%d", argIdx))
		args = append(args, filter.SemesterType)
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

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM academic_periods %s;", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, appErrors.ErrInternal(err)
	}

	allowedSorts := map[string]string{
		"code":        "code",
		"name":        "name",
		"start_date":  "start_date",
		"created_at":  "created_at",
		"-code":       "code DESC",
		"-name":       "name DESC",
		"-start_date": "start_date DESC",
		"-created_at": "created_at DESC",
	}

	orderBy := "start_date DESC"
	if p.Sort != "" {
		if col, ok := allowedSorts[p.Sort]; ok {
			orderBy = col
		}
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, code, name, semester_type, start_date, end_date, is_active, created_at, updated_at
		FROM academic_periods
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

	var result []AcademicPeriod
	for rows.Next() {
		var ap AcademicPeriod
		if err := rows.Scan(
			&ap.ID, &ap.Code, &ap.Name, &ap.SemesterType, &ap.StartDate, &ap.EndDate, &ap.IsActive, &ap.CreatedAt, &ap.UpdatedAt,
		); err != nil {
			return nil, 0, appErrors.ErrInternal(err)
		}
		result = append(result, ap)
	}

	return result, total, nil
}

func (r *pgRepository) Update(ctx context.Context, p *AcademicPeriod) error {
	query := `
		UPDATE academic_periods
		SET name = $2, semester_type = $3, start_date = $4, end_date = $5, is_active = $6, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at;
	`
	err := r.pool.QueryRow(ctx, query,
		p.ID, p.Name, p.SemesterType, p.StartDate, p.EndDate, p.IsActive,
	).Scan(&p.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return appErrors.ErrInternal(err)
	}
	return nil
}
