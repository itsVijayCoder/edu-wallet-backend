package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSProduction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows configured origin", func(t *testing.T) {
		r := gin.New()
		r.Use(CORS("production", []string{"https://app.eduwallet.example"}))
		r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

		req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
		req.Header.Set("Origin", "https://app.eduwallet.example")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.eduwallet.example" {
			t.Fatalf("unexpected allow origin %q", got)
		}
	})

	t.Run("rejects preflight when origins are missing", func(t *testing.T) {
		r := gin.New()
		r.Use(CORS("production", nil))
		r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

		req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
		req.Header.Set("Origin", "https://app.eduwallet.example")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})
}
