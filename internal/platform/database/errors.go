package database

import (
	"errors"

	appErrors "educore/internal/shared/errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// ParseDBError translates PostgreSQL-specific errors into standardized *appErrors.AppError
// without exposing internal table schemas, sensitive SQL queries, or raw connection strings to clients.
func ParseDBError(err error) *appErrors.AppError {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return appErrors.ErrConflict("Data sudah terdaftar atau terjadi pelanggaran keunikan data")
		case pgerrcode.ForeignKeyViolation:
			return appErrors.ErrBadRequest("Relasi data tidak valid atau referensi data tidak ditemukan")
		case pgerrcode.CheckViolation:
			return appErrors.ErrBadRequest("Nilai data tidak memenuhi batasan aturan yang ditentukan")
		case pgerrcode.NotNullViolation:
			return appErrors.ErrBadRequest("Field wajib database tidak boleh bernilai kosong")
		case pgerrcode.AdminShutdown, pgerrcode.CrashShutdown, pgerrcode.CannotConnectNow:
			return appErrors.ErrDependencyUnavailable("Layanan database sedang tidak dapat menerima koneksi", err)
		default:
			return appErrors.ErrInternal(err)
		}
	}

	return appErrors.FromError(err)
}
