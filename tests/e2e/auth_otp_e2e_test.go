package e2e

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth_ParentOTP_E2E(t *testing.T) {
	suite := SetupSuite(t)
	truncateAndReseed(t, suite.Pool, suite.Redis)

	phone := "+919876543210"
	parentID := SeedUser(t, suite.Pool, suite.Hasher, "otp-parent@example.com", "password123", []string{"parents"})
	tenantID := seedOTPGuardian(t, suite, parentID, phone, "greenfield-public-school")

	t.Run("validation and unknown phone", func(t *testing.T) {
		w := doRequest(suite.Server, http.MethodPost, "/api/v1/auth/send-otp", map[string]string{"phone": "9876543210"}, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/send-otp", map[string]string{"phone": "+919999999999"}, nil)
		require.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "AUTH_PHONE_NOT_FOUND", parseJSON(t, w)["error"].(map[string]any)["code"])
	})

	t.Run("send, verify, and consume OTP", func(t *testing.T) {
		w := doRequest(suite.Server, http.MethodPost, "/api/v1/auth/send-otp", map[string]string{"phone": phone}, nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		data := parseJSON(t, w)["data"].(map[string]any)
		assert.Equal(t, "OTP sent to +919876****10", data["message"])
		assert.EqualValues(t, 300, data["expires_in_seconds"])
		otp := suite.OTP.LastOTP(phone)
		require.Len(t, otp, 6)

		w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/verify-otp", map[string]string{"phone": phone, "otp": "000000"}, nil)
		require.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, "AUTH_OTP_INVALID", parseJSON(t, w)["error"].(map[string]any)["code"])

		w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/verify-otp", map[string]string{"phone": phone, "otp": otp}, nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		body := parseJSON(t, w)
		response := body["data"].(map[string]any)
		assert.NotEmpty(t, response["access_token"])
		assert.NotEmpty(t, response["refresh_token"])
		assert.NotEmpty(t, response["expires_at"])
		user := response["user"].(map[string]any)
		assert.Equal(t, parentID.String(), user["id"])
		assert.Equal(t, "parents", user["role"])
		tenants := response["tenants"].([]any)
		require.Len(t, tenants, 1)
		assert.Equal(t, tenantID.String(), tenants[0].(map[string]any)["tenant_id"])

		w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/verify-otp", map[string]string{"phone": phone, "otp": otp}, nil)
		require.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, "AUTH_OTP_EXPIRED", parseJSON(t, w)["error"].(map[string]any)["code"])
	})

	t.Run("send rate limiting", func(t *testing.T) {
		require.NoError(t, suite.Redis.FlushAll(context.Background()).Err())
		w := doRequest(suite.Server, http.MethodPost, "/api/v1/auth/send-otp", map[string]string{"phone": phone}, nil)
		require.Equal(t, http.StatusOK, w.Code)
		w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/send-otp", map[string]string{"phone": phone}, nil)
		require.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, "AUTH_RATE_LIMITED", parseJSON(t, w)["error"].(map[string]any)["code"])
	})

	t.Run("tenant selection is required for multi-tenant phones", func(t *testing.T) {
		secondTenantID := seedOTPGuardian(t, suite, parentID, phone, "another-school")
		w := doRequest(suite.Server, http.MethodPost, "/api/v1/auth/send-otp", map[string]string{"phone": phone}, nil)
		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "AUTH_TENANT_REQUIRED", parseJSON(t, w)["error"].(map[string]any)["code"])

		w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/send-otp", map[string]string{"phone": phone, "tenant_slug": "another-school"}, nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		otp := suite.OTP.LastOTP(phone)
		w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/verify-otp", map[string]string{"phone": phone, "otp": otp}, nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		tenants := parseJSON(t, w)["data"].(map[string]any)["tenants"].([]any)
		assert.Equal(t, secondTenantID.String(), tenants[0].(map[string]any)["tenant_id"])
	})

	t.Run("expired and brute-forced OTPs cannot authenticate", func(t *testing.T) {
		require.NoError(t, suite.Redis.FlushAll(context.Background()).Err())
		w := doRequest(suite.Server, http.MethodPost, "/api/v1/auth/send-otp", map[string]string{"phone": phone, "tenant_slug": "greenfield-public-school"}, nil)
		require.Equal(t, http.StatusOK, w.Code)
		expiredOTP := suite.OTP.LastOTP(phone)
		require.NoError(t, suite.Redis.FlushAll(context.Background()).Err())
		w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/verify-otp", map[string]string{"phone": phone, "otp": expiredOTP}, nil)
		require.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, "AUTH_OTP_EXPIRED", parseJSON(t, w)["error"].(map[string]any)["code"])

		w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/send-otp", map[string]string{"phone": phone, "tenant_slug": "greenfield-public-school"}, nil)
		require.Equal(t, http.StatusOK, w.Code)
		for range 5 {
			w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/verify-otp", map[string]string{"phone": phone, "otp": "000000"}, nil)
			require.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Equal(t, "AUTH_OTP_INVALID", parseJSON(t, w)["error"].(map[string]any)["code"])
		}
		w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/verify-otp", map[string]string{"phone": phone, "otp": "000000"}, nil)
		require.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, "AUTH_OTP_EXPIRED", parseJSON(t, w)["error"].(map[string]any)["code"])
	})
}

func seedOTPGuardian(t *testing.T, suite *TestSuite, userID uuid.UUID, phone, slug string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	var tenantID uuid.UUID
	err := suite.Pool.QueryRow(ctx, `INSERT INTO tenants (name, slug) VALUES ($1, $2) RETURNING id`, slug, slug).Scan(&tenantID)
	require.NoError(t, err)
	_, err = suite.Pool.Exec(ctx, `INSERT INTO guardians (tenant_id, name, relationship, phone, user_id) VALUES ($1, 'OTP Parent', 'parent', $2, $3)`, tenantID, phone, userID)
	require.NoError(t, err)
	return tenantID
}
