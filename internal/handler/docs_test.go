package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func invokeDeployStatus(t *testing.T, h *DocsHandler) map[string]any {
	t.Helper()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/docs/deploy-status", nil)

	h.DeployStatus(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", w.Code, w.Body.String())
	}

	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid deploy-status JSON: %v (body %s)", err, w.Body.String())
	}
	return got
}

func assertBuildFieldsPresent(t *testing.T, got map[string]any) {
	t.Helper()
	build, ok := got["build"].(map[string]any)
	if !ok {
		t.Fatalf("expected build object, got %T", got["build"])
	}
	for _, key := range []string{"sha", "sha_short", "build_time"} {
		if s, _ := build[key].(string); s == "" {
			t.Fatalf("expected build.%s to be non-empty, got %v", key, build[key])
		}
	}
	if _, ok := got["uptime_seconds"]; !ok {
		t.Fatal("expected uptime_seconds in response")
	}
	if s, _ := got["server_time"].(string); s == "" {
		t.Fatal("expected non-empty server_time in response")
	}
}

func TestDeployStatusNoFileReturnsNullDeployments(t *testing.T) {
	h := &DocsHandler{deployStatusFile: ""}

	got := invokeDeployStatus(t, h)

	if v, ok := got["deployments"]; !ok || v != nil {
		t.Fatalf("expected deployments to be null when file unset, got %#v", got["deployments"])
	}
	assertBuildFieldsPresent(t, got)
}

func TestDeployStatusMissingFileReturnsNull(t *testing.T) {
	h := &DocsHandler{deployStatusFile: filepath.Join(t.TempDir(), "absent.json")}

	got := invokeDeployStatus(t, h)

	if got["deployments"] != nil {
		t.Fatalf("expected deployments null for missing file, got %#v", got["deployments"])
	}
	assertBuildFieldsPresent(t, got)
}

func TestDeployStatusInvalidJSONReturnsNull(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	h := &DocsHandler{deployStatusFile: path}

	got := invokeDeployStatus(t, h)

	if got["deployments"] != nil {
		t.Fatalf("expected deployments null for invalid JSON, got %#v", got["deployments"])
	}
}

func TestDeployStatusReadsFixtureFile(t *testing.T) {
	sha := strings.Repeat("a", 40)
	fixture := `{
  "latest_main": {"sha": "` + sha + `", "checked_at": "2026-07-17T10:00:00Z"},
  "held": null,
  "attempts": [
    {
      "id": "20260717-100500",
      "sha": "` + sha + `",
      "sha_short": "aaaaaaaaaaaa",
      "commit_subject": "seed deployment fixture",
      "commit_author": "tester",
      "trigger": "manual",
      "started_at": "2026-07-17T10:05:00Z",
      "finished_at": "2026-07-17T10:06:00Z",
      "duration_s": 60,
      "outcome": "success",
      "phases": [
        {"name": "build", "status": "ok", "duration_s": 42, "detail": "image built"}
      ],
      "error_tail": null
    }
  ]
}`
	path := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	h := &DocsHandler{deployStatusFile: path}

	got := invokeDeployStatus(t, h)

	assertBuildFieldsPresent(t, got)

	deployments, ok := got["deployments"].(map[string]any)
	if !ok {
		t.Fatalf("expected deployments object, got %T (%#v)", got["deployments"], got["deployments"])
	}

	latest, ok := deployments["latest_main"].(map[string]any)
	if !ok {
		t.Fatalf("expected latest_main object, got %T", deployments["latest_main"])
	}
	if latest["sha"] != sha {
		t.Fatalf("expected latest_main.sha %q, got %v", sha, latest["sha"])
	}

	attempts, ok := deployments["attempts"].([]any)
	if !ok || len(attempts) != 1 {
		t.Fatalf("expected exactly 1 attempt, got %#v", deployments["attempts"])
	}
	attempt, ok := attempts[0].(map[string]any)
	if !ok {
		t.Fatalf("expected attempt object, got %T", attempts[0])
	}
	if attempt["outcome"] != "success" {
		t.Fatalf("expected attempt outcome success, got %v", attempt["outcome"])
	}
}

func TestNewDocsHandlerReadsDeployStatusEnv(t *testing.T) {
	t.Setenv("DEPLOY_STATUS_FILE", "/app/deploy-state/status.json")

	h := NewDocsHandler("http://localhost:0")

	if h.deployStatusFile != "/app/deploy-state/status.json" {
		t.Fatalf("expected DEPLOY_STATUS_FILE plumbed into handler, got %q", h.deployStatusFile)
	}
}
