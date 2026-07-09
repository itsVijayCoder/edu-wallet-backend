package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParentsUnifiedView_E2E(t *testing.T) {
	suite := SetupSuite(t)
	truncateAndReseed(t, suite.Pool, suite.Redis)

	adminID := SeedUser(t, suite.Pool, suite.Hasher, "academic-admin@example.com", "password123", []string{"super_admin"})
	tenantAccessToken, tenantID := createTenantSession(t, suite, adminID, "Parents School", "parents-school")

	academicYearID := createAcademicYear(t, suite, tenantAccessToken)
	classID := createClass(t, suite, tenantAccessToken)
	sectionID := createSection(t, suite, tenantAccessToken, academicYearID, classID)
	guardianID := createGuardian(t, suite, tenantAccessToken)
	studentID := createStudent(t, suite, tenantAccessToken, academicYearID, classID, sectionID, guardianID)

	parentUserID := SeedUser(t, suite.Pool, suite.Hasher, "parent@example.com", "parent123", []string{"parents"})

	t.Run("guardian response includes user_id field initially nil", func(t *testing.T) {
		w := doRequest(suite.Server, http.MethodGet, "/api/v1/admin/guardians/"+guardianID.String(), nil, bearer(tenantAccessToken))
		require.Equal(t, http.StatusOK, w.Code)
		body := parseJSON(t, w)
		data := body["data"].(map[string]any)
		assert.Nil(t, data["user_id"])
	})

	t.Run("link guardian to parent user", func(t *testing.T) {
		w := doRequest(suite.Server, http.MethodPost, "/api/v1/admin/guardians/"+guardianID.String()+"/user", map[string]any{
			"user_id": parentUserID.String(),
		}, bearer(tenantAccessToken))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		body := parseJSON(t, w)
		data := body["data"].(map[string]any)
		assert.Equal(t, parentUserID.String(), data["user_id"])
		require.NotNil(t, data["user_status"])
		assert.Equal(t, "active", data["user_status"])
	})

	t.Run("linking same user is idempotent", func(t *testing.T) {
		w := doRequest(suite.Server, http.MethodPost, "/api/v1/admin/guardians/"+guardianID.String()+"/user", map[string]any{
			"user_id": parentUserID.String(),
		}, bearer(tenantAccessToken))
		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("guardian now shows user_id in list and get", func(t *testing.T) {
		w := doRequest(suite.Server, http.MethodGet, "/api/v1/admin/guardians/"+guardianID.String(), nil, bearer(tenantAccessToken))
		require.Equal(t, http.StatusOK, w.Code)
		body := parseJSON(t, w)
		data := body["data"].(map[string]any)
		assert.Equal(t, parentUserID.String(), data["user_id"])
	})

	t.Run("list guardian students (reverse lookup)", func(t *testing.T) {
		w := doRequest(suite.Server, http.MethodGet, "/api/v1/admin/guardians/"+guardianID.String()+"/students", nil, bearer(tenantAccessToken))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		body := parseJSON(t, w)
		data := body["data"].([]any)
		require.Len(t, data, 1)
		student := data[0].(map[string]any)
		assert.Equal(t, studentID.String(), student["student_id"])
		assert.Equal(t, "ADM-100", student["admission_number"])
		assert.Equal(t, "Aarav", student["first_name"])
		assert.Equal(t, "Class 10", student["class_name"])
		assert.Equal(t, "Section A", student["section_name"])
		assert.True(t, student["is_primary"].(bool))
	})

	t.Run("unified parents endpoint", func(t *testing.T) {
		w := doRequest(suite.Server, http.MethodGet, "/api/v1/admin/parents", nil, bearer(tenantAccessToken))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		body := parseJSON(t, w)
		data := body["data"].([]any)
		require.Len(t, data, 1)
		parent := data[0].(map[string]any)
		assert.Equal(t, guardianID.String(), parent["guardian_id"])
		assert.Equal(t, "Riya Sharma", parent["name"])
		assert.Equal(t, parentUserID.String(), parent["user_id"])
		assert.Equal(t, "active", parent["user_status"])
		students := parent["linked_students"].([]any)
		require.Len(t, students, 1)
		assert.Equal(t, studentID.String(), students[0].(map[string]any)["student_id"])
	})

	t.Run("parents endpoint with linked filter", func(t *testing.T) {
		w := doRequest(suite.Server, http.MethodGet, "/api/v1/admin/parents?linked=true", nil, bearer(tenantAccessToken))
		require.Equal(t, http.StatusOK, w.Code)
		body := parseJSON(t, w)
		data := body["data"].([]any)
		require.Len(t, data, 1)
	})

	t.Run("users list filtered by role=parents", func(t *testing.T) {
		w := doRequest(suite.Server, http.MethodGet, "/api/v1/admin/users?role=parents", nil, bearer(tenantAccessToken))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		body := parseJSON(t, w)
		data := body["data"].([]any)
		require.Len(t, data, 1)
		user := data[0].(map[string]any)
		assert.Equal(t, parentUserID.String(), user["id"])
		assert.Contains(t, user["roles"], "parents")
	})

	t.Run("unlink guardian user", func(t *testing.T) {
		w := doRequest(suite.Server, http.MethodDelete, "/api/v1/admin/guardians/"+guardianID.String()+"/user", nil, bearer(tenantAccessToken))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		body := parseJSON(t, w)
		data := body["data"].(map[string]any)
		assert.Nil(t, data["user_id"])

		w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/guardians/"+guardianID.String(), nil, bearer(tenantAccessToken))
		require.Equal(t, http.StatusOK, w.Code)
		body = parseJSON(t, w)
		data = body["data"].(map[string]any)
		assert.Nil(t, data["user_id"])
	})

	t.Run("parents endpoint with unlinked filter after unlink", func(t *testing.T) {
		w := doRequest(suite.Server, http.MethodGet, "/api/v1/admin/parents?linked=false", nil, bearer(tenantAccessToken))
		require.Equal(t, http.StatusOK, w.Code)
		body := parseJSON(t, w)
		data := body["data"].([]any)
		require.Len(t, data, 1)
		assert.Nil(t, data[0].(map[string]any)["user_id"])
	})

	t.Run("linking non-parent user is rejected", func(t *testing.T) {
		nonParentID := SeedUser(t, suite.Pool, suite.Hasher, "staff@example.com", "staff123", []string{"staff"})
		w := doRequest(suite.Server, http.MethodPost, "/api/v1/admin/guardians/"+guardianID.String()+"/user", map[string]any{
			"user_id": nonParentID.String(),
		}, bearer(tenantAccessToken))
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		body := parseJSON(t, w)
		errObj := body["error"].(map[string]any)
		assert.Equal(t, "PARENT_ROLE_MISSING", errObj["code"])
	})

	_ = tenantID
}
