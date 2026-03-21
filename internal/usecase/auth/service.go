package auth

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"eventra/internal/domain/user"
	"eventra/pkg/security"

	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrEmailInUse          = errors.New("email already in use")
	ErrUsernameInUse       = errors.New("username already in use")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrAccountLocked       = errors.New("account is temporarily locked")
	usernameRegex          = regexp.MustCompile(`^[a-zA-Z0-9._-]{3,32}$`)
	emailRegex             = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
)

type Service struct {
	userRepo      UserRepository
	refreshRepo   RefreshTokenRepository
	tokenMgr      TokenManager
	refreshExpiry time.Duration
	securityRepo  LoginSecurityRepository
	auditLogger   AuditLogger
	blacklistRepo TokenBlacklist
}

type ServiceOption func(*Service)

func WithLoginSecurityRepository(repo LoginSecurityRepository) ServiceOption {
	return func(s *Service) { s.securityRepo = repo }
}

func WithAuditLogger(logger AuditLogger) ServiceOption {
	return func(s *Service) { s.auditLogger = logger }
}

func WithTokenBlacklist(repo TokenBlacklist) ServiceOption {
	return func(s *Service) { s.blacklistRepo = repo }
}

func NewService(userRepo UserRepository, refreshRepo RefreshTokenRepository, tokenMgr TokenManager, refreshExpiry time.Duration, opts ...ServiceOption) *Service {
	s := &Service{
		userRepo:      userRepo,
		refreshRepo:   refreshRepo,
		tokenMgr:      tokenMgr,
		refreshExpiry: refreshExpiry,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (AuthResult, error) {
	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	if input.Username == "" || input.Email == "" || input.Password == "" {
		return AuthResult{}, errors.New("username, email and password are required")
	}
	if !usernameRegex.MatchString(input.Username) {
		return AuthResult{}, errors.New("username must be 3-32 chars and only contain letters, numbers, dot, underscore or dash")
	}
	if len(input.Email) > 254 || !emailRegex.MatchString(input.Email) {
		return AuthResult{}, errors.New("invalid email format")
	}
	if len(input.Password) < 8 {
		return AuthResult{}, errors.New("password must be at least 8 characters")
	}

	if _, err := s.userRepo.GetByEmail(ctx, input.Email); err == nil {
		return AuthResult{}, ErrEmailInUse
	} else if !errors.Is(err, user.ErrNotFound) {
		return AuthResult{}, fmt.Errorf("check existing email: %w", err)
	}

	if _, err := s.userRepo.GetByUsername(ctx, input.Username); err == nil {
		return AuthResult{}, ErrUsernameInUse
	} else if !errors.Is(err, user.ErrNotFound) {
		return AuthResult{}, fmt.Errorf("check existing username: %w", err)
	}

	hash, err := security.HashPassword(input.Password)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash password: %w", err)
	}

	newUser := user.User{
		ID:           uuid.New(),
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: hash,
	}

	created, err := s.userRepo.Create(ctx, newUser)
	if err != nil {
		return AuthResult{}, fmt.Errorf("create user: %w", err)
	}

	s.audit(ctx, AuditEvent{
		EventType:  "auth.register.success",
		Severity:   AuditSeverityInfo,
		UserID:     &created.ID,
		IP:         input.ClientIP,
		UserAgent:  input.UserAgent,
		OccurredAt: time.Now().UTC(),
	})

	return s.issueAuthResult(ctx, created)
}

func (s *Service) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if input.Email == "" || input.Password == "" {
		return AuthResult{}, errors.New("email and password are required")
	}

	u, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		s.audit(ctx, AuditEvent{
			EventType:  "auth.login.unknown_email",
			Severity:   AuditSeverityWarn,
			IP:         input.ClientIP,
			UserAgent:  input.UserAgent,
			Metadata:   map[string]any{"email": input.Email},
			OccurredAt: time.Now().UTC(),
		})
		return AuthResult{}, ErrInvalidCredentials
	}

	now := time.Now().UTC()
	state, err := s.getSecurityState(ctx, u.ID)
	if err != nil {
		return AuthResult{}, fmt.Errorf("load security state: %w", err)
	}

	if state.LockedUntil != nil && state.LockedUntil.After(now) {
		s.audit(ctx, AuditEvent{
			EventType:  "auth.login.blocked.locked",
			Severity:   AuditSeverityWarn,
			UserID:     &u.ID,
			IP:         input.ClientIP,
			UserAgent:  input.UserAgent,
			Metadata:   map[string]any{"locked_until": state.LockedUntil.Format(time.RFC3339)},
			OccurredAt: now,
		})
		return AuthResult{}, ErrAccountLocked
	}

	if err = security.CheckPassword(input.Password, u.PasswordHash); err != nil {
		if err = s.recordFailedLogin(ctx, u.ID, input.ClientIP, input.UserAgent, state, now); err != nil {
			return AuthResult{}, fmt.Errorf("record failed login: %w", err)
		}
		return AuthResult{}, ErrInvalidCredentials
	}

	if err = s.recordSuccessfulLogin(ctx, u.ID, input.ClientIP, input.UserAgent, now); err != nil {
		return AuthResult{}, fmt.Errorf("record successful login: %w", err)
	}

	return s.issueAuthResult(ctx, u)
}

func (s *Service) Refresh(ctx context.Context, input RefreshInput) (AuthResult, error) {
	rawRefreshToken := strings.TrimSpace(input.RefreshToken)
	if rawRefreshToken == "" {
		return AuthResult{}, ErrInvalidRefreshToken
	}

	hash := security.HashToken(rawRefreshToken)
	record, err := s.refreshRepo.GetActiveByHash(ctx, hash)
	if err != nil {
		s.audit(ctx, AuditEvent{
			EventType:  "auth.refresh.invalid",
			Severity:   AuditSeverityWarn,
			IP:         input.ClientIP,
			UserAgent:  input.UserAgent,
			OccurredAt: time.Now().UTC(),
		})
		return AuthResult{}, ErrInvalidRefreshToken
	}

	if record.AccessJTI != "" && s.blacklistRepo != nil {
		if err = s.blacklistRepo.BlacklistJTI(ctx, record.AccessJTI, record.AccessExp, "refresh_rotation"); err != nil {
			return AuthResult{}, fmt.Errorf("blacklist rotated token: %w", err)
		}
	}

	u, err := s.userRepo.GetByID(ctx, record.UserID)
	if err != nil {
		return AuthResult{}, ErrInvalidRefreshToken
	}

	if err = s.refreshRepo.RevokeByHash(ctx, hash); err != nil {
		return AuthResult{}, fmt.Errorf("revoke refresh token: %w", err)
	}

	s.audit(ctx, AuditEvent{
		EventType:  "auth.refresh.success",
		Severity:   AuditSeverityInfo,
		UserID:     &u.ID,
		IP:         input.ClientIP,
		UserAgent:  input.UserAgent,
		OccurredAt: time.Now().UTC(),
	})

	return s.issueAuthResult(ctx, u)
}

func (s *Service) Logout(ctx context.Context, input LogoutInput) error {
	rawRefreshToken := strings.TrimSpace(input.RefreshToken)
	if rawRefreshToken == "" {
		return ErrInvalidRefreshToken
	}

	hash := security.HashToken(rawRefreshToken)
	record, err := s.refreshRepo.GetActiveByHash(ctx, hash)
	if err == nil && s.blacklistRepo != nil && record.AccessJTI != "" {
		if blErr := s.blacklistRepo.BlacklistJTI(ctx, record.AccessJTI, record.AccessExp, "logout"); blErr != nil {
			return fmt.Errorf("blacklist token on logout: %w", blErr)
		}
	}

	if err = s.refreshRepo.RevokeByHash(ctx, hash); err != nil {
		return ErrInvalidRefreshToken
	}

	s.audit(ctx, AuditEvent{
		EventType:  "auth.logout.success",
		Severity:   AuditSeverityInfo,
		IP:         input.ClientIP,
		UserAgent:  input.UserAgent,
		OccurredAt: time.Now().UTC(),
	})

	return nil
}

func (s *Service) issueAuthResult(ctx context.Context, u user.User) (AuthResult, error) {
	accessToken, claims, err := s.tokenMgr.GenerateTokenWithClaims(u.ID, u.Email)
	if err != nil {
		return AuthResult{}, fmt.Errorf("generate token: %w", err)
	}

	rawRefreshToken, err := security.GenerateSecureToken(32)
	if err != nil {
		return AuthResult{}, fmt.Errorf("generate refresh token: %w", err)
	}

	refreshRecord := RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: security.HashToken(rawRefreshToken),
		AccessJTI: claims.ID,
		AccessExp: claims.ExpiresAt.Time,
		ExpiresAt: time.Now().UTC().Add(s.refreshExpiry),
	}
	if err = s.refreshRepo.Create(ctx, refreshRecord); err != nil {
		return AuthResult{}, fmt.Errorf("store refresh token: %w", err)
	}

	return AuthResult{
		Token:        accessToken,
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		User:         u,
	}, nil
}

func (s *Service) getSecurityState(ctx context.Context, userID uuid.UUID) (LoginSecurityState, error) {
	if s.securityRepo == nil {
		return LoginSecurityState{UserID: userID}, nil
	}

	state, err := s.securityRepo.GetByUserID(ctx, userID)
	if err != nil {
		return LoginSecurityState{}, err
	}
	if state.UserID == uuid.Nil {
		state.UserID = userID
	}

	return state, nil
}

func (s *Service) recordFailedLogin(ctx context.Context, userID uuid.UUID, clientIP, userAgent string, state LoginSecurityState, now time.Time) error {
	if s.securityRepo == nil {
		return nil
	}

	nextFailures := state.FailedAttempts + 1
	threshold := 5
	if state.LastLoginIP != "" && clientIP != "" && !strings.EqualFold(state.LastLoginIP, clientIP) {
		threshold = 3
	}

	var lockUntil *time.Time
	severity := AuditSeverityWarn
	if nextFailures >= threshold {
		minutes := 5 * (1 << minInt(nextFailures-threshold, 4))
		until := now.Add(time.Duration(minutes) * time.Minute)
		lockUntil = &until
		severity = AuditSeverityHigh
	}

	if err := s.securityRepo.RecordFailure(ctx, userID, nextFailures, clientIP, now, lockUntil); err != nil {
		return err
	}

	metadata := map[string]any{"failed_attempts": nextFailures}
	if lockUntil != nil {
		metadata["locked_until"] = lockUntil.Format(time.RFC3339)
	}

	s.audit(ctx, AuditEvent{
		EventType:  "auth.login.failed",
		Severity:   severity,
		UserID:     &userID,
		IP:         clientIP,
		UserAgent:  userAgent,
		Metadata:   metadata,
		OccurredAt: now,
	})

	return nil
}

func (s *Service) recordSuccessfulLogin(ctx context.Context, userID uuid.UUID, clientIP, userAgent string, now time.Time) error {
	if s.securityRepo != nil {
		if err := s.securityRepo.RecordSuccess(ctx, userID, clientIP, now); err != nil {
			return err
		}
	}

	s.audit(ctx, AuditEvent{
		EventType:  "auth.login.success",
		Severity:   AuditSeverityInfo,
		UserID:     &userID,
		IP:         clientIP,
		UserAgent:  userAgent,
		OccurredAt: now,
	})

	return nil
}

func (s *Service) audit(ctx context.Context, event AuditEvent) {
	if s.auditLogger == nil {
		return
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	_ = s.auditLogger.LogEvent(ctx, event)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
