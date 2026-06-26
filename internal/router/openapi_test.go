package router

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apidoc"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/handler"
	"github.com/itsVijayCoder/edu-wallet-backend/pkg/jwt"
)

func TestOpenAPICatalogCoversRegisteredAPIRoutes(t *testing.T) {
	r := newOpenAPITestRouter()

	registered := map[string]struct{}{}
	for _, route := range r.Routes() {
		if !strings.HasPrefix(route.Path, "/api/v1/") {
			continue
		}
		registered[apidoc.RouteKey(route.Method, route.Path)] = struct{}{}
	}

	documented := map[string]struct{}{}
	for _, ep := range apidoc.Endpoints() {
		documented[apidoc.RouteKey(ep.Method, ep.Path)] = struct{}{}
	}

	var missing []string
	for key := range registered {
		if _, ok := documented[key]; !ok {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)

	var stale []string
	for key := range documented {
		if _, ok := registered[key]; !ok {
			stale = append(stale, key)
		}
	}
	sort.Strings(stale)

	if len(missing) > 0 || len(stale) > 0 {
		t.Fatalf("OpenAPI catalog drift\nmissing routes: %v\nstale routes: %v", missing, stale)
	}
}

func TestDocsEndpointReturnsOpenAPIJSON(t *testing.T) {
	r := newOpenAPITestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/docs/openapi.json", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("expected JSON content type, got %q", w.Header().Get("Content-Type"))
	}

	var spec map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Fatalf("invalid OpenAPI JSON: %v", err)
	}
	if spec["openapi"] != "3.0.3" {
		t.Fatalf("unexpected OpenAPI version: %v", spec["openapi"])
	}
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatal("OpenAPI spec missing paths object")
	}
	if _, ok := paths["/api/v1/auth/login"]; !ok {
		t.Fatal("OpenAPI spec missing auth login path")
	}
}

func TestDocsEndpointReturnsAPITestGuideHTML(t *testing.T) {
	r := newOpenAPITestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/docs/api-test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("expected HTML content type, got %q", w.Header().Get("Content-Type"))
	}
	if !strings.Contains(w.Body.String(), "EduWallet API Test Guide") {
		t.Fatal("expected tester guide title in HTML response")
	}
}

func newOpenAPITestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	log := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	tokenMgr := jwt.NewTokenManager(
		"test-access-secret-that-is-at-least-32-chars!!",
		"test-refresh-secret-that-is-at-least-32-chars!",
		15*time.Minute,
		7*24*time.Hour,
		"test-app",
	)

	return New(log, RouterConfig{
		AppEnv:      "test",
		AppPort:     0,
		ExternalURL: "http://localhost:0",
		CORSOrigins: []string{"*"},
	}, tokenMgr, nil, Handlers{
		Docs:     handler.NewDocsHandler("http://localhost:0"),
		Health:   handler.NewHealthHandler(nil, nil),
		Auth:     handler.NewAuthHandler(nil),
		User:     handler.NewAdminUserHandler(nil),
		Tenant:   handler.NewTenantHandler(nil),
		Academic: handler.NewAcademicHandler(nil),
		Billing:  handler.NewBillingHandler(nil),
		Payment:  handler.NewPaymentHandler(nil),
		Ops:      handler.NewOperationsHandler(nil),
	})
}
