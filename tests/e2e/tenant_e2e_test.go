package e2e

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenant_PlatformCreate_SelectTenant_AdminGet_E2E(t *testing.T) {
	suite := SetupSuite(t)
	truncateAndReseed(t, suite.Pool, suite.Redis)

	adminID := SeedUser(t, suite.Pool, suite.Hasher, "platform@example.com", "password123", []string{"super_admin"})

	loginBody := map[string]string{
		"email":    "platform@example.com",
		"password": "password123",
	}
	w := doRequest(suite.Server, http.MethodPost, "/api/v1/auth/login", loginBody, nil)
	require.Equal(t, http.StatusOK, w.Code)
	platformAccessToken, _ := extractTokens(t, w)

	createTenantBody := map[string]any{
		"name":          "Acme School",
		"slug":          "acme-school",
		"legal_name":    "Acme School Pvt Ltd",
		"contact_email": "admin@acme.test",
		"owner_user_id": adminID.String(),
		"branch": map[string]any{
			"name": "Main Campus",
			"code": "MAIN",
		},
	}
	w = doRequest(suite.Server, http.MethodPost, "/api/v1/platform/tenants", createTenantBody, map[string]string{
		"Authorization": "Bearer " + platformAccessToken,
	})
	require.Equal(t, http.StatusCreated, w.Code)

	body := parseJSON(t, w)
	data := body["data"].(map[string]any)
	tenantID, err := uuid.Parse(data["id"].(string))
	require.NoError(t, err)
	assert.Equal(t, "acme-school", data["slug"])

	var auditCount int
	err = suite.Pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1`, tenantID).Scan(&auditCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, auditCount, 1)

	w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/select-tenant", map[string]string{
		"tenant_id": tenantID.String(),
	}, map[string]string{
		"Authorization": "Bearer " + platformAccessToken,
	})
	require.Equal(t, http.StatusOK, w.Code)
	tenantAccessToken, _ := extractTokens(t, w)

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/tenant", nil, map[string]string{
		"Authorization": "Bearer " + tenantAccessToken,
	})
	require.Equal(t, http.StatusOK, w.Code)

	body = parseJSON(t, w)
	data = body["data"].(map[string]any)
	assert.Equal(t, tenantID.String(), data["id"])
	assert.Equal(t, "Acme School", data["name"])
}
