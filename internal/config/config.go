package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
)

type Config struct {
	App      AppConfig
	Auth     AuthConfig
	DB       database.PostgresConfig
	Redis    database.RedisConfig
	JWT      JWTConfig
	Resend   ResendConfig
	Payments PaymentConfig
}

type AppConfig struct {
	Env                string        `env:"APP_ENV"          envDefault:"development"`
	Mode               string        `env:"APP_MODE"         envDefault:"api"`
	Port               int           `env:"APP_PORT"         envDefault:"8080"`
	Name               string        `env:"APP_NAME"         envDefault:"eduwallet"`
	ExternalURL        string        `env:"APP_EXTERNAL_URL"`
	CORSOrigins        []string      `env:"CORS_ALLOWED_ORIGINS" envSeparator:","`
	WorkerPollInterval time.Duration `env:"WORKER_POLL_INTERVAL" envDefault:"5s"`
}

type AuthConfig struct {
	PublicRegistrationEnabled bool `env:"AUTH_PUBLIC_REGISTRATION_ENABLED" envDefault:"false"`
}

type JWTConfig struct {
	AccessSecret  string        `env:"JWT_ACCESS_SECRET,notEmpty"`
	RefreshSecret string        `env:"JWT_REFRESH_SECRET,notEmpty"`
	AccessExpiry  time.Duration `env:"JWT_ACCESS_EXPIRY"  envDefault:"15m"`
	RefreshExpiry time.Duration `env:"JWT_REFRESH_EXPIRY" envDefault:"168h"`
}

type ResendConfig struct {
	APIKey    string `env:"RESEND_API_KEY"`
	FromEmail string `env:"RESEND_FROM_EMAIL" envDefault:"noreply@example.com"`
	FromName  string `env:"RESEND_FROM_NAME"  envDefault:"eduwallet"`
}

type PaymentConfig struct {
	Provider              string `env:"PAYMENT_PROVIDER" envDefault:"fake"`
	RazorpayKeyID         string `env:"RAZORPAY_KEY_ID"`
	RazorpayKeySecret     string `env:"RAZORPAY_KEY_SECRET"`
	RazorpayWebhookSecret string `env:"RAZORPAY_WEBHOOK_SECRET"`
	RazorpayBaseURL       string `env:"RAZORPAY_BASE_URL" envDefault:"https://api.razorpay.com/v1"`
	FakeSigningSecret     string `env:"PAYMENT_FAKE_SIGNING_SECRET" envDefault:"test_payment_secret"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	if len(cfg.JWT.AccessSecret) < 32 || strings.Contains(cfg.JWT.AccessSecret, "change-me") {
		return fmt.Errorf("JWT_ACCESS_SECRET must be at least 32 characters and not a placeholder")
	}
	if len(cfg.JWT.RefreshSecret) < 32 || strings.Contains(cfg.JWT.RefreshSecret, "change-me") {
		return fmt.Errorf("JWT_REFRESH_SECRET must be at least 32 characters and not a placeholder")
	}
	if cfg.App.Mode != "api" && cfg.App.Mode != "worker" {
		return fmt.Errorf("APP_MODE must be api or worker")
	}
	if cfg.DB.DatabaseURL == "" && cfg.DB.Password == "" {
		return fmt.Errorf("either DATABASE_URL or DB_PASSWORD must be provided")
	}

	if isProduction(cfg.App.Env) {
		if err := validateProduction(cfg); err != nil {
			return err
		}
	}

	return nil
}

func validateProduction(cfg *Config) error {
	if cfg.JWT.AccessSecret == cfg.JWT.RefreshSecret {
		return fmt.Errorf("JWT_ACCESS_SECRET and JWT_REFRESH_SECRET must be different in production")
	}
	if cfg.DB.DatabaseURL == "" && strings.EqualFold(cfg.DB.SSLMode, "disable") {
		return fmt.Errorf("DB_SSL_MODE=disable is not allowed in production when using individual database credentials")
	}
	if cfg.DB.DatabaseURL != "" && strings.Contains(strings.ToLower(cfg.DB.DatabaseURL), "sslmode=disable") {
		return fmt.Errorf("sslmode=disable is not allowed in production DATABASE_URL")
	}
	if strings.TrimSpace(cfg.App.ExternalURL) == "" {
		return fmt.Errorf("APP_EXTERNAL_URL is required in production")
	}
	if err := validatePublicHTTPSURL("APP_EXTERNAL_URL", cfg.App.ExternalURL); err != nil {
		return err
	}
	if len(cfg.App.CORSOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS is required in production")
	}
	validOrigins := 0
	for _, origin := range cfg.App.CORSOrigins {
		origin = strings.TrimSpace(strings.TrimRight(origin, "/"))
		if origin == "" {
			continue
		}
		validOrigins++
		if origin == "*" {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS cannot contain * in production")
		}
		if err := validatePublicHTTPSURL("CORS_ALLOWED_ORIGINS", origin); err != nil {
			return err
		}
	}
	if validOrigins == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS must include at least one origin in production")
	}
	if strings.TrimSpace(cfg.Resend.APIKey) == "" {
		return fmt.Errorf("RESEND_API_KEY is required in production")
	}
	if strings.TrimSpace(cfg.Resend.FromEmail) == "" || strings.Contains(strings.ToLower(cfg.Resend.FromEmail), "example.com") {
		return fmt.Errorf("RESEND_FROM_EMAIL must be a real sender address in production")
	}
	if !strings.EqualFold(cfg.Payments.Provider, "razorpay") {
		return fmt.Errorf("PAYMENT_PROVIDER must be razorpay in production")
	}
	if cfg.Payments.RazorpayKeyID == "" || cfg.Payments.RazorpayKeySecret == "" || cfg.Payments.RazorpayWebhookSecret == "" {
		return fmt.Errorf("razorpay payment provider requires RAZORPAY_KEY_ID, RAZORPAY_KEY_SECRET, and RAZORPAY_WEBHOOK_SECRET")
	}
	if strings.EqualFold(cfg.Payments.RazorpayKeyID, "change-me") ||
		strings.EqualFold(cfg.Payments.RazorpayKeySecret, "change-me") ||
		strings.EqualFold(cfg.Payments.RazorpayWebhookSecret, "change-me") {
		return fmt.Errorf("razorpay credentials cannot be placeholders in production")
	}
	return nil
}

func validatePublicHTTPSURL(name, raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s must be a valid absolute URL", name)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("%s must use https in production", name)
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasPrefix(host, "127.") || host == "::1" {
		return fmt.Errorf("%s cannot use localhost in production", name)
	}
	return nil
}

func isProduction(env string) bool {
	return strings.EqualFold(strings.TrimSpace(env), "production")
}
