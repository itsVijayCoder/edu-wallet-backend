package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	pg  *pgxpool.Pool
	rdb *redis.Client
}

func NewHealthHandler(pg *pgxpool.Pool, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{pg: pg, rdb: rdb}
}

// Healthz is a liveness probe. It always returns 200 if the process is running.
func (h *HealthHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readyz is a readiness probe. It pings Postgres and Redis, measures latency,
// and returns 503 if any dependency is unreachable.
func (h *HealthHandler) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	overall := "healthy"
	statusCode := http.StatusOK

	// Check Postgres.
	pgStatus, pgLatency := h.pingPostgres(ctx)

	// Check Redis.
	redisStatus, redisLatency := h.pingRedis(ctx)

	if pgStatus != "up" || redisStatus != "up" {
		overall = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status": overall,
		"dependencies": gin.H{
			"postgres": gin.H{
				"status":     pgStatus,
				"latency_ms": pgLatency,
			},
			"redis": gin.H{
				"status":     redisStatus,
				"latency_ms": redisLatency,
			},
		},
	})
}

func (h *HealthHandler) pingPostgres(ctx context.Context) (string, int64) {
	start := time.Now()
	if err := h.pg.Ping(ctx); err != nil {
		return "down", 0
	}
	return "up", time.Since(start).Milliseconds()
}

func (h *HealthHandler) pingRedis(ctx context.Context) (string, int64) {
	start := time.Now()
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		return "down", 0
	}
	return "up", time.Since(start).Milliseconds()
}
