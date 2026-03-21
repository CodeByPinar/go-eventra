package postgres

import (
	"context"
	"errors"
	"fmt"

	"eventra/internal/usecase/auth"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRefreshTokenNotFound = errors.New("refresh token not found")

type RefreshTokenRepository struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepository(db *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token auth.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, access_jti, access_expires_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	if _, err := r.db.Exec(ctx, query, token.ID, token.UserID, token.TokenHash, token.AccessJTI, token.AccessExp, token.ExpiresAt); err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepository) GetActiveByHash(ctx context.Context, tokenHash string) (auth.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, access_jti, access_expires_at, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`

	var token auth.RefreshToken
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.AccessJTI,
		&token.AccessExp,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.RefreshToken{}, ErrRefreshTokenNotFound
	}
	if err != nil {
		return auth.RefreshToken{}, fmt.Errorf("get refresh token by hash: %w", err)
	}

	return token, nil
}

func (r *RefreshTokenRepository) RevokeByHash(ctx context.Context, tokenHash string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1
		  AND revoked_at IS NULL
	`

	res, err := r.db.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrRefreshTokenNotFound
	}

	return nil
}
