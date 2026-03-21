package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"eventra/internal/domain/user"
	"eventra/pkg/security"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type stubUserRepo struct {
	createFn        func(context.Context, user.User) (user.User, error)
	getByEmailFn    func(context.Context, string) (user.User, error)
	getByUsernameFn func(context.Context, string) (user.User, error)
	getByIDFn       func(context.Context, uuid.UUID) (user.User, error)
}

func (s stubUserRepo) Create(ctx context.Context, u user.User) (user.User, error) {
	return s.createFn(ctx, u)
}

func (s stubUserRepo) GetByEmail(ctx context.Context, email string) (user.User, error) {
	return s.getByEmailFn(ctx, email)
}

func (s stubUserRepo) GetByUsername(ctx context.Context, username string) (user.User, error) {
	return s.getByUsernameFn(ctx, username)
}

func (s stubUserRepo) GetByID(ctx context.Context, id uuid.UUID) (user.User, error) {
	if s.getByIDFn == nil {
		return user.User{}, user.ErrNotFound
	}
	return s.getByIDFn(ctx, id)
}

type stubRefreshRepo struct {
	createFn          func(context.Context, RefreshToken) error
	getActiveByHashFn func(context.Context, string) (RefreshToken, error)
	revokeByHashFn    func(context.Context, string) error
}

func (s stubRefreshRepo) Create(ctx context.Context, token RefreshToken) error {
	return s.createFn(ctx, token)
}

func (s stubRefreshRepo) GetActiveByHash(ctx context.Context, tokenHash string) (RefreshToken, error) {
	return s.getActiveByHashFn(ctx, tokenHash)
}

func (s stubRefreshRepo) RevokeByHash(ctx context.Context, tokenHash string) error {
	return s.revokeByHashFn(ctx, tokenHash)
}

type stubTokenManager struct {
	generateTokenFn           func(uuid.UUID, string) (string, error)
	generateTokenWithClaimsFn func(uuid.UUID, string) (string, security.Claims, error)
}

type stubLoginSecurityRepo struct {
	state         LoginSecurityState
	recordFailure func(context.Context, uuid.UUID, int, string, time.Time, *time.Time) error
	recordSuccess func(context.Context, uuid.UUID, string, time.Time) error
}

func (s *stubLoginSecurityRepo) GetByUserID(context.Context, uuid.UUID) (LoginSecurityState, error) {
	return s.state, nil
}

func (s *stubLoginSecurityRepo) RecordFailure(ctx context.Context, userID uuid.UUID, failedAttempts int, ip string, now time.Time, lockUntil *time.Time) error {
	if s.recordFailure != nil {
		return s.recordFailure(ctx, userID, failedAttempts, ip, now, lockUntil)
	}
	s.state.UserID = userID
	s.state.FailedAttempts = failedAttempts
	s.state.LockedUntil = lockUntil
	s.state.LastLoginIP = ip
	return nil
}

func (s *stubLoginSecurityRepo) RecordSuccess(ctx context.Context, userID uuid.UUID, ip string, now time.Time) error {
	if s.recordSuccess != nil {
		return s.recordSuccess(ctx, userID, ip, now)
	}
	s.state.UserID = userID
	s.state.FailedAttempts = 0
	s.state.LockedUntil = nil
	s.state.LastLoginIP = ip
	return nil
}

func (s stubTokenManager) GenerateToken(userID uuid.UUID, email string) (string, error) {
	if s.generateTokenFn != nil {
		return s.generateTokenFn(userID, email)
	}
	if s.generateTokenWithClaimsFn != nil {
		token, _, err := s.generateTokenWithClaimsFn(userID, email)
		return token, err
	}
	return "", errors.New("not implemented")
}

func (s stubTokenManager) GenerateTokenWithClaims(userID uuid.UUID, email string) (string, security.Claims, error) {
	if s.generateTokenWithClaimsFn != nil {
		return s.generateTokenWithClaimsFn(userID, email)
	}
	if s.generateTokenFn != nil {
		token, err := s.generateTokenFn(userID, email)
		if err != nil {
			return "", security.Claims{}, err
		}
		now := time.Now().UTC()
		return token, security.Claims{RegisteredClaims: jwt.RegisteredClaims{ID: uuid.NewString(), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}}, nil
	}
	return "", security.Claims{}, errors.New("not implemented")
}

func TestRegisterSuccess(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Now().UTC()

	var saved user.User
	repo := stubUserRepo{
		createFn: func(_ context.Context, u user.User) (user.User, error) {
			saved = u
			u.CreatedAt = createdAt
			return u, nil
		},
		getByEmailFn: func(context.Context, string) (user.User, error) {
			return user.User{}, user.ErrNotFound
		},
		getByUsernameFn: func(context.Context, string) (user.User, error) {
			return user.User{}, user.ErrNotFound
		},
	}

	tokens := stubTokenManager{
		generateTokenFn: func(_ uuid.UUID, _ string) (string, error) {
			return "token-123", nil
		},
	}
	refresh := stubRefreshRepo{
		createFn: func(context.Context, RefreshToken) error { return nil },
		getActiveByHashFn: func(context.Context, string) (RefreshToken, error) {
			return RefreshToken{}, errors.New("should not be called")
		},
		revokeByHashFn: func(context.Context, string) error {
			return errors.New("should not be called")
		},
	}

	svc := NewService(repo, refresh, tokens, 7*24*time.Hour)
	out, err := svc.Register(ctx, RegisterInput{
		Username: "  alice  ",
		Email:    "ALICE@EXAMPLE.COM",
		Password: "StrongPass123!",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if out.Token != "token-123" {
		t.Fatalf("expected token-123, got %s", out.Token)
	}
	if out.RefreshToken == "" {
		t.Fatalf("expected refresh token to be set")
	}
	if saved.Username != "alice" {
		t.Fatalf("expected trimmed username alice, got %q", saved.Username)
	}
	if saved.Email != "alice@example.com" {
		t.Fatalf("expected normalized email alice@example.com, got %q", saved.Email)
	}
	if err = security.CheckPassword("StrongPass123!", saved.PasswordHash); err != nil {
		t.Fatalf("expected password hash to match, got %v", err)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	ctx := context.Background()
	repo := stubUserRepo{
		createFn: func(context.Context, user.User) (user.User, error) {
			return user.User{}, errors.New("should not be called")
		},
		getByEmailFn: func(context.Context, string) (user.User, error) {
			return user.User{ID: uuid.New()}, nil
		},
		getByUsernameFn: func(context.Context, string) (user.User, error) {
			return user.User{}, user.ErrNotFound
		},
	}
	tokens := stubTokenManager{generateTokenFn: func(uuid.UUID, string) (string, error) { return "", nil }}
	refresh := stubRefreshRepo{
		createFn: func(context.Context, RefreshToken) error { return nil },
		getActiveByHashFn: func(context.Context, string) (RefreshToken, error) {
			return RefreshToken{}, errors.New("should not be called")
		},
		revokeByHashFn: func(context.Context, string) error { return nil },
	}

	svc := NewService(repo, refresh, tokens, 7*24*time.Hour)
	_, err := svc.Register(ctx, RegisterInput{Username: "alice", Email: "alice@example.com", Password: "StrongPass123!"})
	if !errors.Is(err, ErrEmailInUse) {
		t.Fatalf("expected ErrEmailInUse, got %v", err)
	}
}

func TestLoginInvalidPassword(t *testing.T) {
	ctx := context.Background()
	hash, err := security.HashPassword("StrongPass123!")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	repo := stubUserRepo{
		createFn: func(context.Context, user.User) (user.User, error) {
			return user.User{}, errors.New("should not be called")
		},
		getByEmailFn: func(context.Context, string) (user.User, error) {
			return user.User{ID: uuid.New(), Email: "alice@example.com", PasswordHash: hash}, nil
		},
		getByUsernameFn: func(context.Context, string) (user.User, error) {
			return user.User{}, user.ErrNotFound
		},
	}
	tokens := stubTokenManager{generateTokenFn: func(uuid.UUID, string) (string, error) { return "", nil }}
	refresh := stubRefreshRepo{
		createFn: func(context.Context, RefreshToken) error { return nil },
		getActiveByHashFn: func(context.Context, string) (RefreshToken, error) {
			return RefreshToken{}, errors.New("should not be called")
		},
		revokeByHashFn: func(context.Context, string) error { return nil },
	}

	svc := NewService(repo, refresh, tokens, 7*24*time.Hour)
	_, err = svc.Login(ctx, LoginInput{Email: "alice@example.com", Password: "WrongPass123!"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginSuccess(t *testing.T) {
	ctx := context.Background()
	hash, err := security.HashPassword("StrongPass123!")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	userID := uuid.New()
	repo := stubUserRepo{
		createFn: func(context.Context, user.User) (user.User, error) {
			return user.User{}, errors.New("should not be called")
		},
		getByEmailFn: func(context.Context, string) (user.User, error) {
			return user.User{ID: userID, Email: "alice@example.com", PasswordHash: hash, Username: "alice"}, nil
		},
		getByUsernameFn: func(context.Context, string) (user.User, error) {
			return user.User{}, user.ErrNotFound
		},
	}
	tokens := stubTokenManager{
		generateTokenFn: func(gotID uuid.UUID, gotEmail string) (string, error) {
			if gotID != userID || gotEmail != "alice@example.com" {
				t.Fatalf("unexpected token payload id=%s email=%s", gotID, gotEmail)
			}
			return "token-abc", nil
		},
	}
	refresh := stubRefreshRepo{
		createFn: func(context.Context, RefreshToken) error { return nil },
		getActiveByHashFn: func(context.Context, string) (RefreshToken, error) {
			return RefreshToken{}, errors.New("should not be called")
		},
		revokeByHashFn: func(context.Context, string) error { return nil },
	}

	svc := NewService(repo, refresh, tokens, 7*24*time.Hour)
	out, err := svc.Login(ctx, LoginInput{Email: " ALICE@EXAMPLE.COM ", Password: "StrongPass123!"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Token != "token-abc" {
		t.Fatalf("expected token-abc, got %s", out.Token)
	}
	if out.RefreshToken == "" {
		t.Fatalf("expected refresh token to be set")
	}
}

func TestRefreshSuccess(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	repo := stubUserRepo{
		createFn: func(context.Context, user.User) (user.User, error) {
			return user.User{}, errors.New("should not be called")
		},
		getByEmailFn: func(context.Context, string) (user.User, error) {
			return user.User{}, user.ErrNotFound
		},
		getByUsernameFn: func(context.Context, string) (user.User, error) {
			return user.User{}, user.ErrNotFound
		},
		getByIDFn: func(context.Context, uuid.UUID) (user.User, error) {
			return user.User{ID: userID, Email: "alice@example.com", Username: "alice"}, nil
		},
	}

	refreshRepo := stubRefreshRepo{
		createFn: func(context.Context, RefreshToken) error { return nil },
		getActiveByHashFn: func(context.Context, string) (RefreshToken, error) {
			return RefreshToken{UserID: userID}, nil
		},
		revokeByHashFn: func(context.Context, string) error { return nil },
	}

	tokens := stubTokenManager{generateTokenFn: func(uuid.UUID, string) (string, error) { return "new-access", nil }}

	svc := NewService(repo, refreshRepo, tokens, 7*24*time.Hour)
	out, err := svc.Refresh(ctx, RefreshInput{RefreshToken: "refresh-token"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.AccessToken != "new-access" || out.RefreshToken == "" {
		t.Fatalf("expected rotated tokens, got %+v", out)
	}
}

func TestLogoutInvalidRefreshToken(t *testing.T) {
	ctx := context.Background()
	repo := stubUserRepo{
		createFn: func(context.Context, user.User) (user.User, error) {
			return user.User{}, errors.New("should not be called")
		},
		getByEmailFn: func(context.Context, string) (user.User, error) {
			return user.User{}, user.ErrNotFound
		},
		getByUsernameFn: func(context.Context, string) (user.User, error) {
			return user.User{}, user.ErrNotFound
		},
	}
	refreshRepo := stubRefreshRepo{
		createFn: func(context.Context, RefreshToken) error { return nil },
		getActiveByHashFn: func(context.Context, string) (RefreshToken, error) {
			return RefreshToken{}, errors.New("should not be called")
		},
		revokeByHashFn: func(context.Context, string) error { return errors.New("not found") },
	}
	tokens := stubTokenManager{generateTokenFn: func(uuid.UUID, string) (string, error) { return "", nil }}

	svc := NewService(repo, refreshRepo, tokens, 7*24*time.Hour)
	err := svc.Logout(ctx, LogoutInput{RefreshToken: "refresh-token"})
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected ErrInvalidRefreshToken, got %v", err)
	}
}

func TestLoginAuthAbuseLockoutPolicy(t *testing.T) {
	ctx := context.Background()
	hash, err := security.HashPassword("StrongPass123!")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	userID := uuid.New()
	repo := stubUserRepo{
		createFn: func(context.Context, user.User) (user.User, error) {
			return user.User{}, errors.New("should not be called")
		},
		getByEmailFn: func(context.Context, string) (user.User, error) {
			return user.User{ID: userID, Email: "alice@example.com", PasswordHash: hash}, nil
		},
		getByUsernameFn: func(context.Context, string) (user.User, error) { return user.User{}, user.ErrNotFound },
	}

	securityRepo := &stubLoginSecurityRepo{}
	refreshRepo := stubRefreshRepo{
		createFn: func(context.Context, RefreshToken) error { return nil },
		getActiveByHashFn: func(context.Context, string) (RefreshToken, error) {
			return RefreshToken{}, errors.New("should not be called")
		},
		revokeByHashFn: func(context.Context, string) error { return nil },
	}
	tokens := stubTokenManager{generateTokenFn: func(uuid.UUID, string) (string, error) { return "token", nil }}

	svc := NewService(repo, refreshRepo, tokens, 7*24*time.Hour, WithLoginSecurityRepository(securityRepo))

	for i := 0; i < 5; i++ {
		_, _ = svc.Login(ctx, LoginInput{Email: "alice@example.com", Password: "wrong", ClientIP: "198.51.100.12"})
	}

	_, err = svc.Login(ctx, LoginInput{Email: "alice@example.com", Password: "wrong", ClientIP: "198.51.100.12"})
	if !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("expected ErrAccountLocked after repeated failures, got %v", err)
	}
}
