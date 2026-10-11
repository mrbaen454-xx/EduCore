package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"educore/internal/platform/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database interactions for users, roles, and refresh tokens.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new instance of Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetUserByEmail retrieves a user and their assigned roles by normalized email address.
// Returns nil, nil if no matching user is found.
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))

	queryUser := `
		SELECT id, email, password_hash, name, is_active, created_at, updated_at
		FROM users
		WHERE LOWER(TRIM(email)) = $1
		LIMIT 1;
	`

	var u User
	err := r.pool.QueryRow(ctx, queryUser, cleanEmail).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Name,
		&u.IsActive,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query user by email: %w", err)
	}

	// Fetch assigned role IDs
	roles, err := r.getUserRoles(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	u.Roles = roles

	return &u, nil
}

// GetUserByID retrieves a user and their assigned roles by unique user ID.
// Returns nil, nil if no matching user is found.
func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	queryUser := `
		SELECT id, email, password_hash, name, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
		LIMIT 1;
	`

	var u User
	err := r.pool.QueryRow(ctx, queryUser, id).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Name,
		&u.IsActive,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query user by id: %w", err)
	}

	roles, err := r.getUserRoles(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	u.Roles = roles

	return &u, nil
}

// getUserRoles retrieves all role IDs assigned to a user.
func (r *Repository) getUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	queryRoles := `
		SELECT role_id
		FROM user_roles
		WHERE user_id = $1
		ORDER BY role_id ASC;
	`

	rows, err := r.pool.Query(ctx, queryRoles, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user roles: %w", err)
	}
	defer rows.Close()

	roles := make([]string, 0)
	for rows.Next() {
		var roleID string
		if err := rows.Scan(&roleID); err != nil {
			return nil, fmt.Errorf("failed to scan user role: %w", err)
		}
		roles = append(roles, roleID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user roles: %w", err)
	}

	return roles, nil
}

// CreateUser inserts a user and their initial roles in an atomic transaction.
func (r *Repository) CreateUser(ctx context.Context, u *User, roles []string) error {
	return database.RunInTx(ctx, r.pool, func(tx pgx.Tx) error {
		queryUser := `
			INSERT INTO users (id, email, password_hash, name, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7);
		`

		if u.ID == uuid.Nil {
			u.ID = uuid.New()
		}

		_, err := tx.Exec(ctx, queryUser,
			u.ID,
			strings.ToLower(strings.TrimSpace(u.Email)),
			u.PasswordHash,
			u.Name,
			u.IsActive,
			u.CreatedAt,
			u.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
		}

		queryRole := `
			INSERT INTO user_roles (user_id, role_id)
			VALUES ($1, $2)
			ON CONFLICT (user_id, role_id) DO NOTHING;
		`

		for _, roleID := range roles {
			if _, err := tx.Exec(ctx, queryRole, u.ID, roleID); err != nil {
				return fmt.Errorf("failed to assign role %s to user: %w", roleID, err)
			}
		}

		return nil
	})
}

// CreateRefreshToken persists a new refresh token record.
func (r *Repository) CreateRefreshToken(ctx context.Context, rt *RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked_at, replaced_by, user_agent, client_ip, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);
	`

	if rt.ID == uuid.Nil {
		rt.ID = uuid.New()
	}

	_, err := r.pool.Exec(ctx, query,
		rt.ID,
		rt.UserID,
		rt.TokenHash,
		rt.ExpiresAt,
		rt.RevokedAt,
		rt.ReplacedBy,
		rt.UserAgent,
		rt.ClientIP,
		rt.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert refresh token: %w", err)
	}

	return nil
}

// GetRefreshTokenByHash retrieves a refresh token record by its SHA-256 hash.
// Returns nil, nil if not found.
func (r *Repository) GetRefreshTokenByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked_at, replaced_by, user_agent, client_ip, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
		LIMIT 1;
	`

	var rt RefreshToken
	err := r.pool.QueryRow(ctx, query, hash).Scan(
		&rt.ID,
		&rt.UserID,
		&rt.TokenHash,
		&rt.ExpiresAt,
		&rt.RevokedAt,
		&rt.ReplacedBy,
		&rt.UserAgent,
		&rt.ClientIP,
		&rt.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query refresh token by hash: %w", err)
	}

	return &rt, nil
}

// RevokeRefreshToken marks a specific refresh token as revoked.
func (r *Repository) RevokeRefreshToken(ctx context.Context, id uuid.UUID, replacedBy *uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = CURRENT_TIMESTAMP, replaced_by = $2
		WHERE id = $1 AND revoked_at IS NULL;
	`

	_, err := r.pool.Exec(ctx, query, id, replacedBy)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

// RevokeAllUserRefreshTokens revokes all active refresh tokens belonging to a user (used on reuse detection or security reset).
func (r *Repository) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND revoked_at IS NULL;
	`

	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke all refresh tokens for user: %w", err)
	}

	return nil
}

// RotateRefreshToken atomically revokes an old refresh token and stores the new refresh token in a single transaction.
func (r *Repository) RotateRefreshToken(ctx context.Context, oldTokenID uuid.UUID, newRT *RefreshToken) error {
	return database.RunInTx(ctx, r.pool, func(tx pgx.Tx) error {
		if newRT.ID == uuid.Nil {
			newRT.ID = uuid.New()
		}

		// 1. Insert new refresh token first so foreign key constraint replaced_by is satisfied
		queryInsert := `
			INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked_at, replaced_by, user_agent, client_ip, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);
		`
		_, err := tx.Exec(ctx, queryInsert,
			newRT.ID,
			newRT.UserID,
			newRT.TokenHash,
			newRT.ExpiresAt,
			newRT.RevokedAt,
			newRT.ReplacedBy,
			newRT.UserAgent,
			newRT.ClientIP,
			newRT.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert rotated refresh token: %w", err)
		}

		// 2. Revoke old token and link to new replacement
		queryRevoke := `
			UPDATE refresh_tokens
			SET revoked_at = CURRENT_TIMESTAMP, replaced_by = $2
			WHERE id = $1 AND revoked_at IS NULL;
		`
		res, err := tx.Exec(ctx, queryRevoke, oldTokenID, newRT.ID)
		if err != nil {
			return fmt.Errorf("failed to revoke old refresh token during rotation: %w", err)
		}
		if res.RowsAffected() == 0 {
			return fmt.Errorf("refresh token was already revoked or not found")
		}

		return nil
	})
}
