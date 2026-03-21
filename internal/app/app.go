package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"eventra/internal/config"
	"eventra/internal/delivery/httpserver"
	"eventra/internal/repository/postgres"
	"eventra/internal/usecase/auth"
	"eventra/pkg/database"
	"eventra/pkg/security"
)

func Run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	dbPool, err := database.NewPostgresPool(ctx, cfg.DBURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer dbPool.Close()

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
	authHandler := httpserver.NewAuthHandler(authService)

	router := httpserver.NewRouter(authHandler, jwtManager, tokenBlacklistRepo, cfg.CORSAllowedOrigins)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("eventra auth service listening on :%s", cfg.Port)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
		close(errCh)
	}()

	shutdownCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-shutdownCtx.Done():
		ctxTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err = server.Shutdown(ctxTimeout); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		return nil
	case serveErr := <-errCh:
		if serveErr != nil {
			return fmt.Errorf("http server: %w", serveErr)
		}
		return nil
	}
}
