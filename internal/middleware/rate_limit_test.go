package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func TestRateLimitFailsClosedWhenRedisIsUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rdb := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  10 * time.Millisecond,
		ReadTimeout:  10 * time.Millisecond,
		WriteTimeout: 10 * time.Millisecond,
		MaxRetries:   0,
	})
	t.Cleanup(func() { _ = rdb.Close() })

	r := gin.New()
	r.POST("/login", RateLimit(rdb, 5, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/login", nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when Redis is unavailable, got %d", w.Code)
	}
	if got := w.Body.String(); got == "" || !strings.Contains(got, "RATE_LIMIT_UNAVAILABLE") {
		t.Fatalf("expected RATE_LIMIT_UNAVAILABLE response, got %q", got)
	}
}
