package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocsOpenAPI_E2E(t *testing.T) {
	suite := SetupSuite(t)

	w := doRequest(suite.Server, http.MethodGet, "/api/v1/docs/openapi.json", nil, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var spec map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &spec))
	assert.Equal(t, "3.0.3", spec["openapi"])

	paths, ok := spec["paths"].(map[string]any)
	require.True(t, ok, "expected paths object")
	assert.Contains(t, paths, "/api/v1/parent/payments/orders")
	assert.Contains(t, paths, "/api/v1/webhooks/razorpay")
}
