package e2e

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcademicStudentImport_E2E(t *testing.T) {
	suite := SetupSuite(t)
	truncateAndReseed(t, suite.Pool, suite.Redis)

	adminID := SeedUser(t, suite.Pool, suite.Hasher, "academic-admin@example.com", "password123", []string{"super_admin"})
	tenantAccessToken, tenantID := createTenantSession(t, suite, adminID, "Acme School", "acme-academic")

	academicYearID := createAcademicYear(t, suite, tenantAccessToken)
	classID := createClass(t, suite, tenantAccessToken)
	sectionID := createSection(t, suite, tenantAccessToken, academicYearID, classID)
	guardianID := createGuardian(t, suite, tenantAccessToken)
	studentID := createStudent(t, suite, tenantAccessToken, academicYearID, classID, sectionID, guardianID)

	w := doRequest(suite.Server, http.MethodGet, "/api/v1/admin/students?status=active&search=aarav", nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body := parseJSON(t, w)
	data := body["data"].([]any)
	require.Len(t, data, 1)
	assert.Equal(t, studentID.String(), data[0].(map[string]any)["id"])

	csvPayload := fmt.Sprintf(`admission_number,first_name,last_name,academic_year_code,class_code,section_code,roll_number,status,category,guardian_name,guardian_relationship,guardian_phone,guardian_email
ADM-200,Meera,Iyer,2026-27,10,A,8,active,general,Kavya Iyer,mother,9000000001,kavya@example.test
`)
	w = doRequest(suite.Server, http.MethodPost, "/api/v1/admin/imports/students/preview", map[string]any{
		"filename": "students.csv",
		"csv":      csvPayload,
	}, bearer(tenantAccessToken))
	require.Equal(t, http.StatusCreated, w.Code)
	body = parseJSON(t, w)
	preview := body["data"].(map[string]any)
	require.Equal(t, float64(1), preview["valid_rows"])
	require.Equal(t, float64(0), preview["invalid_rows"])
	importID, err := uuid.Parse(preview["import_id"].(string))
	require.NoError(t, err)

	w = doRequest(suite.Server, http.MethodPost, "/api/v1/admin/imports/students/commit", map[string]string{
		"import_id": importID.String(),
	}, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	commitData := body["data"].(map[string]any)
	assert.Equal(t, float64(1), commitData["committed_rows"])

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/students?search=meera", nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	data = body["data"].([]any)
	require.Len(t, data, 1)
	assert.Equal(t, "ADM-200", data[0].(map[string]any)["admission_number"])

	var auditCount int
	err = suite.Pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND action IN ('student.created', 'student_import.committed')`, tenantID).Scan(&auditCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, auditCount, 2)

	otherToken, _ := createTenantSession(t, suite, adminID, "Other School", "other-academic")
	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/students/"+studentID.String(), nil, bearer(otherToken))
	require.Equal(t, http.StatusNotFound, w.Code)
}

func createTenantSession(t *testing.T, suite *TestSuite, ownerID uuid.UUID, name, slug string) (string, uuid.UUID) {
	t.Helper()
	w := doRequest(suite.Server, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    "academic-admin@example.com",
		"password": "password123",
	}, nil)
	require.Equal(t, http.StatusOK, w.Code)
	platformAccessToken, _ := extractTokens(t, w)

	w = doRequest(suite.Server, http.MethodPost, "/api/v1/platform/tenants", map[string]any{
		"name":          name,
		"slug":          slug,
		"legal_name":    name + " Pvt Ltd",
		"contact_email": "admin@" + slug + ".test",
		"owner_user_id": ownerID.String(),
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
	tenantAccessToken, _ := extractTokens(t, w)
	return tenantAccessToken, tenantID
}

func createAcademicYear(t *testing.T, suite *TestSuite, token string) uuid.UUID {
	t.Helper()
	w := doRequest(suite.Server, http.MethodPost, "/api/v1/admin/academic-years", map[string]any{
		"name":       "Academic Year 2026-27",
		"code":       "2026-27",
		"start_date": "2026-04-01",
		"end_date":   "2027-03-31",
		"is_active":  true,
	}, bearer(token))
	require.Equal(t, http.StatusCreated, w.Code)
	return responseID(t, w)
}

func createClass(t *testing.T, suite *TestSuite, token string) uuid.UUID {
	t.Helper()
	w := doRequest(suite.Server, http.MethodPost, "/api/v1/admin/classes", map[string]any{
		"name":       "Class 10",
		"code":       "10",
		"sort_order": 10,
	}, bearer(token))
	require.Equal(t, http.StatusCreated, w.Code)
	return responseID(t, w)
}

func createSection(t *testing.T, suite *TestSuite, token string, academicYearID, classID uuid.UUID) uuid.UUID {
	t.Helper()
	w := doRequest(suite.Server, http.MethodPost, "/api/v1/admin/sections", map[string]any{
		"academic_year_id": academicYearID.String(),
		"class_id":         classID.String(),
		"name":             "Section A",
		"code":             "A",
		"capacity":         40,
	}, bearer(token))
	require.Equal(t, http.StatusCreated, w.Code)
	return responseID(t, w)
}

func createGuardian(t *testing.T, suite *TestSuite, token string) uuid.UUID {
	t.Helper()
	w := doRequest(suite.Server, http.MethodPost, "/api/v1/admin/guardians", map[string]any{
		"name":                 "Riya Sharma",
		"relationship":         "mother",
		"phone":                "9000000000",
		"email":                "riya.sharma@example.test",
		"preferred_language":   "en",
		"communication_opt_in": true,
	}, bearer(token))
	require.Equal(t, http.StatusCreated, w.Code)
	return responseID(t, w)
}

func createStudent(t *testing.T, suite *TestSuite, token string, academicYearID, classID, sectionID, guardianID uuid.UUID) uuid.UUID {
	t.Helper()
	w := doRequest(suite.Server, http.MethodPost, "/api/v1/admin/students", map[string]any{
		"academic_year_id":      academicYearID.String(),
		"class_id":              classID.String(),
		"section_id":            sectionID.String(),
		"admission_number":      "ADM-100",
		"first_name":            "Aarav",
		"last_name":             "Sharma",
		"roll_number":           "7",
		"status":                "active",
		"category":              "general",
		"opening_balance_paise": 125000,
		"guardians": []map[string]any{
			{
				"guardian_id":  guardianID.String(),
				"relationship": "mother",
				"is_primary":   true,
			},
		},
	}, bearer(token))
	require.Equal(t, http.StatusCreated, w.Code)
	body := parseJSON(t, w)
	data := body["data"].(map[string]any)
	guardians := data["guardians"].([]any)
	require.Len(t, guardians, 1)
	assert.Equal(t, guardianID.String(), guardians[0].(map[string]any)["guardian_id"])
	return responseID(t, w)
}

func responseID(t *testing.T, w *httptest.ResponseRecorder) uuid.UUID {
	t.Helper()
	body := parseJSON(t, w)
	data := body["data"].(map[string]any)
	id, err := uuid.Parse(data["id"].(string))
	require.NoError(t, err)
	return id
}

func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}
