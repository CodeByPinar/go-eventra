package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenBlacklistRepository struct {
	db *pgxpool.Pool
}

func NewTokenBlacklistRepository(db *pgxpool.Pool) *TokenBlacklistRepository {
	return &TokenBlacklistRepository{db: db}
}

func (r *TokenBlacklistRepository) BlacklistJTI(ctx context.Context, jti string, expiresAt time.Time, reason string) error {
	query := `
		INSERT INTO revoked_jtis (jti, expires_at, reason, revoked_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (jti)
		DO NOTHING
	`

	if _, err := r.db.Exec(ctx, query, jti, expiresAt, reason); err != nil {
		return fmt.Errorf("blacklist jti: %w", err)
	}

	return nil
}

func (r *TokenBlacklistRepository) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	query := `
		SELECT 1
		FROM revoked_jtis
		WHERE jti = $1
		  AND expires_at > NOW()
	`

	var one int
	err := r.db.QueryRow(ctx, query, jti).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check blacklisted jti: %w", err)
	}

	return true, nil
}
