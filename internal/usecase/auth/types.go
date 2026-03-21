package auth

import (
	"context"
	"time"

	"eventra/internal/domain/user"
	"eventra/pkg/security"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(context.Context, user.User) (user.User, error)
	GetByEmail(context.Context, string) (user.User, error)
	GetByUsername(context.Context, string) (user.User, error)
	GetByID(context.Context, uuid.UUID) (user.User, error)
}

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	AccessJTI string
	AccessExp time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type RefreshTokenRepository interface {
	Create(context.Context, RefreshToken) error
	GetActiveByHash(context.Context, string) (RefreshToken, error)
	RevokeByHash(context.Context, string) error
}

type TokenManager interface {
	GenerateToken(userID uuid.UUID, email string) (string, error)
	GenerateTokenWithClaims(userID uuid.UUID, email string) (string, security.Claims, error)
}

type LoginSecurityState struct {
	UserID         uuid.UUID
	FailedAttempts int
	LastLoginIP    string
	LockedUntil    *time.Time
}

type LoginSecurityRepository interface {
	GetByUserID(context.Context, uuid.UUID) (LoginSecurityState, error)
	RecordFailure(context.Context, uuid.UUID, int, string, time.Time, *time.Time) error
	RecordSuccess(context.Context, uuid.UUID, string, time.Time) error
}

type AuditSeverity string

const (
	AuditSeverityInfo AuditSeverity = "info"
	AuditSeverityWarn AuditSeverity = "warn"
	AuditSeverityHigh AuditSeverity = "high"
)

type AuditEvent struct {
	EventType  string
	Severity   AuditSeverity
	UserID     *uuid.UUID
	IP         string
	UserAgent  string
	Metadata   map[string]any
	OccurredAt time.Time
}

type AuditLogger interface {
	LogEvent(context.Context, AuditEvent) error
}

type TokenBlacklist interface {
	BlacklistJTI(context.Context, string, time.Time, string) error
	IsBlacklisted(context.Context, string) (bool, error)
}

type RegisterInput struct {
	Username  string
	Email     string
	Password  string
	ClientIP  string
	UserAgent string
}

type LoginInput struct {
	Email     string
	Password  string
	ClientIP  string
	UserAgent string
}

type RefreshInput struct {
	RefreshToken string
	ClientIP     string
	UserAgent    string
}

type LogoutInput struct {
	RefreshToken string
	ClientIP     string
	UserAgent    string
}

type AuthResult struct {
	Token        string
	AccessToken  string
	RefreshToken string
	User         user.User
}
