package database

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	RedisURL string `env:"REDIS_URL"`
	Host     string `env:"REDIS_HOST"     envDefault:"localhost"`
	Port     int    `env:"REDIS_PORT"     envDefault:"6379"`
	Password string `env:"REDIS_PASSWORD"`
	DB       int    `env:"REDIS_DB"       envDefault:"0"`
}

func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func NewRedisClient(ctx context.Context, cfg RedisConfig, log *slog.Logger) (*redis.Client, error) {
	var opts *redis.Options
	var addr string

	if cfg.RedisURL != "" {
		parsed, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			return nil, fmt.Errorf("parse redis url: %w", err)
		}
		opts = parsed
		addr = opts.Addr
	} else {
		opts = &redis.Options{
			Addr:     cfg.Addr(),
			Password: cfg.Password,
			DB:       cfg.DB,
		}
		addr = cfg.Addr()
	}

	rdb := redis.NewClient(opts)

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	log.Info("connected to Redis", "addr", addr)
	return rdb, nil
}
