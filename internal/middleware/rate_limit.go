package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimit returns a middleware that enforces a sliding window rate limit
// per client IP and route using Redis.
func RateLimit(rdb *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		key := fmt.Sprintf("rl:%s:%s", c.ClientIP(), c.FullPath())

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			// Rate-limited routes include authentication and payment mutations. Failing
			// open here would remove their brute-force and abuse protection exactly
			// when the limiter dependency is unhealthy.
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_UNAVAILABLE",
					"message": "request limiting is temporarily unavailable",
				},
			})
			return
		}

		// Set expiry only on the first request in the window.
		if count == 1 {
			if err := rdb.Expire(ctx, key, window).Err(); err != nil {
				_ = rdb.Del(ctx, key).Err()
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "RATE_LIMIT_UNAVAILABLE",
						"message": "request limiting is temporarily unavailable",
					},
				})
				return
			}
		}

		remaining := int64(limit) - count
		if remaining < 0 {
			remaining = 0
		}

		resetAt := time.Now().Add(window).Unix()
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))

		if count > int64(limit) {
			retryAfter := int(window.Seconds())
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "too many requests, please try again later",
				},
			})
			return
		}

		c.Next()
	}
}
