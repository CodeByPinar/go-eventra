package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"eventra/internal/usecase/auth"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LoginSecurityRepository struct {
	db *pgxpool.Pool
}

func NewLoginSecurityRepository(db *pgxpool.Pool) *LoginSecurityRepository {
	return &LoginSecurityRepository{db: db}
}

func (r *LoginSecurityRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (auth.LoginSecurityState, error) {
	query := `
		SELECT user_id, failed_attempts, COALESCE(last_login_ip, ''), locked_until
		FROM account_security_state
		WHERE user_id = $1
	`

	var state auth.LoginSecurityState
	err := r.db.QueryRow(ctx, query, userID).Scan(&state.UserID, &state.FailedAttempts, &state.LastLoginIP, &state.LockedUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.LoginSecurityState{UserID: userID}, nil
	}
	if err != nil {
		return auth.LoginSecurityState{}, fmt.Errorf("get login security state: %w", err)
	}

	return state, nil
}

func (r *LoginSecurityRepository) RecordFailure(ctx context.Context, userID uuid.UUID, failedAttempts int, ip string, now time.Time, lockUntil *time.Time) error {
	query := `
		INSERT INTO account_security_state (user_id, failed_attempts, last_failed_at, last_login_ip, locked_until, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id)
		DO UPDATE SET
			failed_attempts = EXCLUDED.failed_attempts,
			last_failed_at = EXCLUDED.last_failed_at,
			last_login_ip = EXCLUDED.last_login_ip,
			locked_until = EXCLUDED.locked_until,
			updated_at = NOW()
	`

	if _, err := r.db.Exec(ctx, query, userID, failedAttempts, now, ip, lockUntil); err != nil {
		return fmt.Errorf("record failed login: %w", err)
	}

	return nil
}

func (r *LoginSecurityRepository) RecordSuccess(ctx context.Context, userID uuid.UUID, ip string, now time.Time) error {
	query := `
		INSERT INTO account_security_state (user_id, failed_attempts, last_login_at, last_login_ip, locked_until, updated_at)
		VALUES ($1, 0, $2, $3, NULL, NOW())
		ON CONFLICT (user_id)
		DO UPDATE SET
			failed_attempts = 0,
			last_login_at = EXCLUDED.last_login_at,
			last_login_ip = EXCLUDED.last_login_ip,
			locked_until = NULL,
			updated_at = NOW()
	`

	if _, err := r.db.Exec(ctx, query, userID, now, ip); err != nil {
		return fmt.Errorf("record successful login: %w", err)
	}

	return nil
}
