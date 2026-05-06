package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

// doRequest builds an HTTP request, serialises the body as JSON (if non-nil),
// sets headers, and records the response through the test gin.Engine.
func doRequest(server *gin.Engine, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = &bytes.Buffer{}
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

// parseJSON is a convenience to unmarshal a recorder body into a map.
func parseJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &m)
	require.NoError(t, err)
	return m
}

// extractTokens pulls access_token and refresh_token out of the standard
// APIResponse data envelope.
func extractTokens(t *testing.T, w *httptest.ResponseRecorder) (accessToken, refreshToken string) {
	t.Helper()
	body := parseJSON(t, w)
	data, ok := body["data"].(map[string]any)
	require.True(t, ok, "expected data object in response")
	accessToken, _ = data["access_token"].(string)
	refreshToken, _ = data["refresh_token"].(string)
	require.NotEmpty(t, accessToken, "access_token should not be empty")
	require.NotEmpty(t, refreshToken, "refresh_token should not be empty")
	return
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestAuth_Register_Login_E2E(t *testing.T) {
	suite := SetupSuite(t)
	truncateAndReseed(t, suite.Pool, suite.Redis)

	// --- Register ---
	regBody := map[string]string{
		"email":      "newuser@example.com",
		"password":   "securepass123",
		"first_name": "New",
		"last_name":  "User",
	}
	w := doRequest(suite.Server, http.MethodPost, "/api/v1/auth/register", regBody, nil)
	assert.Equal(t, http.StatusCreated, w.Code)

	regResp := parseJSON(t, w)
	assert.True(t, regResp["success"].(bool))
	data := regResp["data"].(map[string]any)
	assert.Equal(t, "newuser@example.com", data["email"])

	// --- Login ---
	loginBody := map[string]string{
		"email":    "newuser@example.com",
		"password": "securepass123",
	}
	w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/login", loginBody, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	accessToken, refreshToken := extractTokens(t, w)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
}

func TestAuth_Refresh_E2E(t *testing.T) {
	suite := SetupSuite(t)
	truncateAndReseed(t, suite.Pool, suite.Redis)

	// Seed a user and login.
	SeedUser(t, suite.Pool, suite.Hasher, "refresh@example.com", "password123", []string{"member"})

	loginBody := map[string]string{
		"email":    "refresh@example.com",
		"password": "password123",
	}
	w := doRequest(suite.Server, http.MethodPost, "/api/v1/auth/login", loginBody, nil)
	require.Equal(t, http.StatusOK, w.Code)

	_, refreshToken := extractTokens(t, w)

	// --- Refresh ---
	refreshBody := map[string]string{"refresh_token": refreshToken}
	w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/refresh", refreshBody, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	newAccess, newRefresh := extractTokens(t, w)
	assert.NotEmpty(t, newAccess)
	assert.NotEmpty(t, newRefresh)
	// The new tokens should differ from the originals (rotation).
	assert.NotEqual(t, refreshToken, newRefresh)
}

func TestAuth_Logout_E2E(t *testing.T) {
	suite := SetupSuite(t)
	truncateAndReseed(t, suite.Pool, suite.Redis)

	SeedUser(t, suite.Pool, suite.Hasher, "logout@example.com", "password123", []string{"member"})

	// Login.
	loginBody := map[string]string{
		"email":    "logout@example.com",
		"password": "password123",
	}
	w := doRequest(suite.Server, http.MethodPost, "/api/v1/auth/login", loginBody, nil)
	require.Equal(t, http.StatusOK, w.Code)

	accessToken, refreshToken := extractTokens(t, w)

	// --- Logout ---
	w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/logout", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	assert.Equal(t, http.StatusOK, w.Code)

	// --- Verify refresh token is now invalid ---
	refreshBody := map[string]string{"refresh_token": refreshToken}
	w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/refresh", refreshBody, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_InvalidToken_E2E(t *testing.T) {
	suite := SetupSuite(t)
	truncateAndReseed(t, suite.Pool, suite.Redis)

	// Hit the logout endpoint (which requires auth) with a bad token.
	w := doRequest(suite.Server, http.MethodPost, "/api/v1/auth/logout", nil, map[string]string{
		"Authorization": "Bearer totally-invalid-token",
	})
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	body := parseJSON(t, w)
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok, "expected error object")
	assert.Equal(t, "AUTH_INVALID_TOKEN", errObj["code"])
}

func TestAuth_ForgotPassword_E2E(t *testing.T) {
	suite := SetupSuite(t)
	truncateAndReseed(t, suite.Pool, suite.Redis)

	// Forgot-password should always return 200 regardless of whether the
	// email exists, to prevent email enumeration.

	// Case 1: existing user
	SeedUser(t, suite.Pool, suite.Hasher, "forgot@example.com", "password123", []string{"member"})
	w := doRequest(suite.Server, http.MethodPost, "/api/v1/auth/forgot-password",
		map[string]string{"email": "forgot@example.com"}, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	// Case 2: non-existent email
	w = doRequest(suite.Server, http.MethodPost, "/api/v1/auth/forgot-password",
		map[string]string{"email": "noone@example.com"}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}
