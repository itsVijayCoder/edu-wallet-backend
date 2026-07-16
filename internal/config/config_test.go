package config

import (
	"strings"
	"testing"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
)

func TestValidateProductionConfig(t *testing.T) {
	t.Parallel()

	t.Run("accepts strict production settings", func(t *testing.T) {
		t.Parallel()
		cfg := productionConfig()

		if err := validate(cfg); err != nil {
			t.Fatalf("validate production config: %v", err)
		}
	})

	t.Run("requires distinct JWT secrets", func(t *testing.T) {
		t.Parallel()
		cfg := productionConfig()
		cfg.JWT.RefreshSecret = cfg.JWT.AccessSecret

		err := validate(cfg)
		if err == nil || !strings.Contains(err.Error(), "must be different") {
			t.Fatalf("expected distinct JWT secret error, got %v", err)
		}
	})

	t.Run("rejects wildcard CORS", func(t *testing.T) {
		t.Parallel()
		cfg := productionConfig()
		cfg.App.CORSOrigins = []string{"*"}

		err := validate(cfg)
		if err == nil || !strings.Contains(err.Error(), "CORS_ALLOWED_ORIGINS") {
			t.Fatalf("expected CORS error, got %v", err)
		}
	})

	t.Run("rejects blank CORS origins", func(t *testing.T) {
		t.Parallel()
		cfg := productionConfig()
		cfg.App.CORSOrigins = []string{"", " "}

		err := validate(cfg)
		if err == nil || !strings.Contains(err.Error(), "CORS_ALLOWED_ORIGINS") {
			t.Fatalf("expected CORS error, got %v", err)
		}
	})

	t.Run("rejects fake payment provider", func(t *testing.T) {
		t.Parallel()
		cfg := productionConfig()
		cfg.Payments.Provider = "fake"

		err := validate(cfg)
		if err == nil || !strings.Contains(err.Error(), "PAYMENT_PROVIDER") {
			t.Fatalf("expected payment provider error, got %v", err)
		}
	})

	t.Run("requires public HTTPS URL", func(t *testing.T) {
		t.Parallel()
		cfg := productionConfig()
		cfg.App.ExternalURL = "http://localhost:8080"

		err := validate(cfg)
		if err == nil || !strings.Contains(err.Error(), "APP_EXTERNAL_URL") {
			t.Fatalf("expected external URL error, got %v", err)
		}
	})

	t.Run("rejects weak super admin bootstrap password", func(t *testing.T) {
		t.Parallel()
		cfg := productionConfig()
		cfg.Auth.SuperAdminBootstrapEnabled = true
		cfg.Auth.SuperAdminEmail = "admin@eduwallet.in"
		cfg.Auth.SuperAdminPassword = "password"

		err := validate(cfg)
		if err == nil || !strings.Contains(err.Error(), "SUPER_ADMIN_PASSWORD") {
			t.Fatalf("expected super admin password error, got %v", err)
		}
	})

	t.Run("rejects malformed trusted proxy", func(t *testing.T) {
		t.Parallel()
		cfg := productionConfig()
		cfg.App.TrustedProxies = []string{"not-an-ip"}

		err := validate(cfg)
		if err == nil || !strings.Contains(err.Error(), "TRUSTED_PROXIES") {
			t.Fatalf("expected trusted proxy error, got %v", err)
		}
	})
}

func productionConfig() *Config {
	return &Config{
		App: AppConfig{
			Env:         "production",
			Mode:        "api",
			ExternalURL: "https://api.eduwallet.example",
			CORSOrigins: []string{"https://app.eduwallet.example"},
		},
		DB: database.PostgresConfig{
			Host:     "db.example",
			Port:     5432,
			User:     "eduwallet",
			Password: "strong-db-password",
			Name:     "eduwallet",
			SSLMode:  "require",
		},
		JWT: JWTConfig{
			AccessSecret:  "access-secret-that-is-at-least-thirty-two-characters",
			RefreshSecret: "refresh-secret-that-is-at-least-thirty-two-characters",
		},
		Resend: ResendConfig{
			APIKey:    "re_test_key",
			FromEmail: "noreply@eduwallet.example",
			FromName:  "EduWallet",
		},
		Payments: PaymentConfig{
			Provider:              "razorpay",
			RazorpayKeyID:         "rzp_test_key",
			RazorpayKeySecret:     "rzp_secret",
			RazorpayWebhookSecret: "webhook_secret",
		},
	}
}
