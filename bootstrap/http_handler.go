package bootstrap

import (
	"context"
	"fmt"
	"net/http"

	"eventra/internal/config"
	"eventra/internal/delivery/httpserver"
	"eventra/internal/repository/postgres"
	"eventra/internal/usecase/auth"
	usecaseevent "eventra/internal/usecase/event"
	"eventra/pkg/database"
	"eventra/pkg/security"
)

func NewHTTPHandler(ctx context.Context) (http.Handler, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	dbPool, err := database.NewPostgresPool(ctx, cfg.DBURL)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	jwtManager := security.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiration)
	userRepo := postgres.NewUserRepository(dbPool)
	refreshRepo := postgres.NewRefreshTokenRepository(dbPool)
	securityRepo := postgres.NewLoginSecurityRepository(dbPool)
	auditRepo := postgres.NewSecurityAuditRepository(dbPool, cfg.SecurityAlertWebhookURL, cfg.SecurityAlertWebhookFormat)
	tokenBlacklistRepo := postgres.NewTokenBlacklistRepository(dbPool)

	authService := auth.NewService(
		userRepo,
		refreshRepo,
		jwtManager,
		cfg.RefreshTokenExpiration,
		auth.WithLoginSecurityRepository(securityRepo),
		auth.WithAuditLogger(auditRepo),
		auth.WithTokenBlacklist(tokenBlacklistRepo),
	)
	eventService := usecaseevent.NewService(postgres.NewEventRepository(dbPool))

	authHandler := httpserver.NewAuthHandler(authService)
	eventHandler := httpserver.NewEventHandler(eventService)
	router := httpserver.NewRouter(authHandler, eventHandler, jwtManager, tokenBlacklistRepo, cfg.CORSAllowedOrigins)
	return router, nil
}
