package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/config"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/handler"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository/postgres"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/router"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/service"
	"github.com/itsVijayCoder/edu-wallet-backend/pkg/email"
	"github.com/itsVijayCoder/edu-wallet-backend/pkg/hasher"
	"github.com/itsVijayCoder/edu-wallet-backend/pkg/jwt"
	"github.com/itsVijayCoder/edu-wallet-backend/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// --- Configuration ---
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// --- Logger ---
	log := logger.New(cfg.App.Env)
	log.Info("starting eduwallet", "env", cfg.App.Env, "port", cfg.App.Port)

	// --- Databases ---
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPostgresPool(ctx, cfg.DB, log)
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}

	rdb, err := database.NewRedisClient(ctx, cfg.Redis, log)
	if err != nil {
		return fmt.Errorf("redis: %w", err)
	}

	// --- Packages ---
	h := hasher.NewBcryptHasher(12)
	tokenMgr := jwt.NewTokenManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
		cfg.App.Name,
	)
	emailClient := email.NewClient(cfg.Resend.APIKey, cfg.Resend.FromEmail, cfg.Resend.FromName)
	emailSvc := service.NewEmailService(emailClient, cfg.App.ExternalURL, log)

	// --- Repositories ---
	roleRepo := postgres.NewRoleRepository(pool)
	userRepo := postgres.NewUserRepository(pool)
	sessionRepo := postgres.NewSessionRepository(pool)
	_ = sessionRepo // used for session cleanup; wire into services as needed
	tenantRepo := postgres.NewTenantRepository(pool)
	membershipRepo := postgres.NewTenantMembershipRepository(pool)
	auditRepo := postgres.NewAuditRepository(pool)
	academicRepo := postgres.NewAcademicRepository(pool)
	transactor := database.NewTransactor(pool)
	// --- ADD YOUR REPOSITORIES HERE ---

	// --- Services ---
	publicRegistrationEnabled := cfg.Auth.PublicRegistrationEnabled || cfg.App.Env != "production"
	authSvc := service.NewAuthService(
		userRepo,
		h,
		tokenMgr,
		rdb,
		cfg.JWT.RefreshExpiry,
		emailSvc,
		log,
		publicRegistrationEnabled,
		membershipRepo,
	)
	userSvc := service.NewUserService(userRepo, roleRepo, h, rdb)
	tenantSvc := service.NewTenantService(tenantRepo, membershipRepo, roleRepo, auditRepo)
	academicSvc := service.NewAcademicService(academicRepo, postgres.NewAcademicRepository, transactor, auditRepo)
	// --- ADD YOUR SERVICES HERE ---

	// --- Router ---
	r := router.New(log, router.RouterConfig{
		AppEnv:      cfg.App.Env,
		AppPort:     cfg.App.Port,
		ExternalURL: cfg.App.ExternalURL,
		CORSOrigins: cfg.App.CORSOrigins,
	}, tokenMgr, rdb, router.Handlers{
		Health:   handler.NewHealthHandler(pool, rdb),
		Auth:     handler.NewAuthHandler(authSvc),
		User:     handler.NewAdminUserHandler(userSvc),
		Tenant:   handler.NewTenantHandler(tenantSvc),
		Academic: handler.NewAcademicHandler(academicSvc),
		// --- ADD YOUR HANDLERS HERE ---
	})

	// --- HTTP Server ---
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// --- Graceful Shutdown ---
	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Info("received shutdown signal", "signal", sig.String())
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("forced shutdown", slog.String("error", err.Error()))
	}

	pool.Close()
	if err := rdb.Close(); err != nil {
		log.Error("redis close error", slog.String("error", err.Error()))
	}

	log.Info("eduwallet stopped")
	return nil
}
