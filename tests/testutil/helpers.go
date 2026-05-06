package testutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/handler"
)

// NewTestGinContext creates a fresh gin.Context backed by an httptest.ResponseRecorder.
// Useful for unit-testing handlers in isolation.
func NewTestGinContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

// SetAuthContext injects the same context keys that the Auth middleware sets,
// allowing handler tests to run without going through the middleware.
func SetAuthContext(c *gin.Context, userID uuid.UUID, email string, roles []string) {
	c.Set("user_id", userID.String())
	c.Set("user_email", email)
	c.Set("user_roles", roles)
}

// ParseResponse unmarshals the JSON body into an APIResponse and extracts the
// Data field into the generic type T.
func ParseResponse[T any](t *testing.T, body *bytes.Buffer) T {
	t.Helper()

	var envelope struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	err := json.Unmarshal(body.Bytes(), &envelope)
	require.NoError(t, err, "failed to unmarshal API envelope")
	require.True(t, envelope.Success, "expected success=true in API response")

	var data T
	err = json.Unmarshal(envelope.Data, &data)
	require.NoError(t, err, "failed to unmarshal data field")
	return data
}

// AssertAPIError validates that the response body contains a failure envelope
// with the expected machine-readable error code.
func AssertAPIError(t *testing.T, body *bytes.Buffer, expectedCode string) {
	t.Helper()

	var envelope struct {
		Success bool              `json:"success"`
		Error   *handler.APIError `json:"error"`
	}
	err := json.Unmarshal(body.Bytes(), &envelope)
	require.NoError(t, err, "failed to unmarshal API error envelope")
	require.False(t, envelope.Success, "expected success=false in error response")
	require.NotNil(t, envelope.Error, "expected error object in response")
	require.Equal(t, expectedCode, envelope.Error.Code, "unexpected error code")
}
