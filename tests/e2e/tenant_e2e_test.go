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

func TestTenant_SuperAdminAccessAndTenantAdminUserCreate_E2E(t *testing.T) {
	suite := SetupSuite(t)
	truncateAndReseed(t, suite.Pool, suite.Redis)

	SeedUser(t, suite.Pool, suite.Hasher, "owner@example.com", "password123", []string{"super_admin"})
	schoolAdminID := SeedUser(t, suite.Pool, suite.Hasher, "school-admin@example.com", "password123", nil)

	w := doRequest(suite.Server, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    "owner@example.com",
		"password": "password123",
	}, nil)
	require.Equal(t, http.StatusOK, w.Code)
	platformAccessToken, _ := extractTokens(t, w)

	w = doRequest(suite.Server, http.MethodPost, "/api/v1/platform/tenants", map[string]any{
		"name":          "Independent School",
		"slug":          "independent-school",
		"legal_name":    "Independent School",
		"contact_email": "admin@independent.test",
		"owner_user_id": schoolAdminID.String(),
		"branch": map[string]any{
			"name": "Main Campus",
			"code": "MAIN",
		},
	}, bearer(platformAccessToken))
	require.Equal(t, http.StatusCreated, w.Code)

	body := parseJSON(t, w)
	data := body["data"].(map[string]any)
	tenantID, err := uuid.Parse(data["id"].(string))
	require.NoError(t, err)

	w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/select-tenant", map[string]string{
		"tenant_id": tenantID.String(),
	}, bearer(platformAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	superTenantToken, _ := extractTokens(t, w)

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/tenant", nil, bearer(superTenantToken))
	require.Equal(t, http.StatusOK, w.Code)

	w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    "school-admin@example.com",
		"password": "password123",
	}, nil)
	require.Equal(t, http.StatusOK, w.Code)
	schoolAdminToken, _ := extractTokens(t, w)

	w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/select-tenant", map[string]string{
		"tenant_id": tenantID.String(),
	}, bearer(schoolAdminToken))
	require.Equal(t, http.StatusOK, w.Code)
	tenantAdminToken, _ := extractTokens(t, w)

	w = doRequest(suite.Server, http.MethodPost, "/api/v1/admin/tenant/users", map[string]string{
		"email":      "fee-staff@example.com",
		"password":   "password123",
		"first_name": "Fee",
		"last_name":  "Staff",
		"role":       "staff",
	}, bearer(tenantAdminToken))
	require.Equal(t, http.StatusCreated, w.Code)

	body = parseJSON(t, w)
	data = body["data"].(map[string]any)
	assert.Equal(t, tenantID.String(), data["tenant_id"])
	assert.Equal(t, "staff", data["tenant_role"])

	user := data["user"].(map[string]any)
	assert.Equal(t, "fee-staff@example.com", user["email"])

	var membershipCount int
	err = suite.Pool.QueryRow(t.Context(), `
		SELECT COUNT(*)
		FROM tenant_memberships tm
		JOIN users u ON u.id = tm.user_id
		JOIN roles r ON r.id = tm.role_id
		WHERE tm.tenant_id = $1
		  AND u.email = 'fee-staff@example.com'
		  AND r.slug = 'staff'
		  AND tm.status = 'active'
	`, tenantID).Scan(&membershipCount)
	require.NoError(t, err)
	assert.Equal(t, 1, membershipCount)
}
