package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresConfig struct {
	DatabaseURL     string        `env:"DATABASE_URL"`
	Host            string        `env:"DB_HOST"              envDefault:"localhost"`
	Port            int           `env:"DB_PORT"              envDefault:"5432"`
	User            string        `env:"DB_USER"              envDefault:"postgres"`
	Password        string        `env:"DB_PASSWORD"`
	Name            string        `env:"DB_NAME"              envDefault:"app_db"`
	SSLMode         string        `env:"DB_SSL_MODE"          envDefault:"disable"`
	MaxOpenConns    int32         `env:"DB_MAX_OPEN_CONNS"    envDefault:"25"`
	MaxIdleConns    int32         `env:"DB_MAX_IDLE_CONNS"    envDefault:"5"`
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"5m"`
}

func (c PostgresConfig) DSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}

func NewPostgresPool(ctx context.Context, cfg PostgresConfig, log *slog.Logger) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxOpenConns
	poolCfg.MinConns = cfg.MaxIdleConns
	poolCfg.MaxConnLifetime = cfg.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	log.Info("connected to PostgreSQL", "host", cfg.Host, "db", cfg.Name)
	return pool, nil
}
