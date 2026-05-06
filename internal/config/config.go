package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
)

type Config struct {
	App    AppConfig
	DB     database.PostgresConfig
	Redis  database.RedisConfig
	JWT    JWTConfig
	Resend ResendConfig
}

type AppConfig struct {
	Env         string   `env:"APP_ENV"          envDefault:"development"`
	Port        int      `env:"APP_PORT"         envDefault:"8080"`
	Name        string   `env:"APP_NAME"         envDefault:"eduwallet"`
	ExternalURL string   `env:"APP_EXTERNAL_URL"`
	CORSOrigins []string `env:"CORS_ALLOWED_ORIGINS" envSeparator:","`
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
	// JWT secret entropy check
	if len(cfg.JWT.AccessSecret) < 32 || strings.Contains(cfg.JWT.AccessSecret, "change-me") {
		return fmt.Errorf("JWT_ACCESS_SECRET must be at least 32 characters and not a placeholder")
	}
	if len(cfg.JWT.RefreshSecret) < 32 || strings.Contains(cfg.JWT.RefreshSecret, "change-me") {
		return fmt.Errorf("JWT_REFRESH_SECRET must be at least 32 characters and not a placeholder")
	}

	// SSL enforcement in production
	if cfg.App.Env == "production" && cfg.DB.SSLMode == "disable" {
		return fmt.Errorf("DB_SSL_MODE=disable is not allowed in production")
	}

	return nil
}
