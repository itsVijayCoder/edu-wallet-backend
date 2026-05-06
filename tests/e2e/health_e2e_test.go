package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthz_ReturnsOK(t *testing.T) {
	suite := SetupSuite(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	w := httptest.NewRecorder()
	suite.Server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "ok", body["status"])
}

func TestReadyz_ReturnsDependencyStatus(t *testing.T) {
	suite := SetupSuite(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/readyz", nil)
	w := httptest.NewRecorder()
	suite.Server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)

	assert.Equal(t, "healthy", body["status"])

	deps, ok := body["dependencies"].(map[string]any)
	require.True(t, ok, "dependencies should be an object")

	pgDep, ok := deps["postgres"].(map[string]any)
	require.True(t, ok, "postgres dependency should be an object")
	assert.Equal(t, "up", pgDep["status"])

	redisDep, ok := deps["redis"].(map[string]any)
	require.True(t, ok, "redis dependency should be an object")
	assert.Equal(t, "up", redisDep["status"])
}
