package e2e

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBillingFeeSetupInvoiceLedger_E2E(t *testing.T) {
	suite := SetupSuite(t)
	truncateAndReseed(t, suite.Pool, suite.Redis)

	adminID := SeedUser(t, suite.Pool, suite.Hasher, "academic-admin@example.com", "password123", []string{"super_admin"})
	tenantAccessToken, tenantID := createTenantSession(t, suite, adminID, "Billing School", "billing-school")

	academicYearID := createAcademicYear(t, suite, tenantAccessToken)
	classID := createClass(t, suite, tenantAccessToken)
	sectionID := createSection(t, suite, tenantAccessToken, academicYearID, classID)
	guardianID := createGuardian(t, suite, tenantAccessToken)
	studentID := createStudent(t, suite, tenantAccessToken, academicYearID, classID, sectionID, guardianID)

	tuitionHeadID := createFeeHead(t, suite, tenantAccessToken, "Tuition Fee", "TUITION", "tuition")
	examHeadID := createFeeHead(t, suite, tenantAccessToken, "Exam Fee", "EXAM", "exam")
	feeStructureID := createFeeStructure(t, suite, tenantAccessToken, academicYearID, tuitionHeadID, examHeadID)
	assignmentID := createFeeAssignment(t, suite, tenantAccessToken, feeStructureID, classID)

	w := doRequest(suite.Server, http.MethodPost, "/api/v1/admin/invoices/generate", map[string]any{
		"assignment_id":        assignmentID.String(),
		"issue_date":           "2026-06-01",
		"due_date":             "2026-06-10",
		"billing_period_start": "2026-06-01",
		"billing_period_end":   "2026-06-30",
	}, bearer(tenantAccessToken))
	require.Equal(t, http.StatusCreated, w.Code)
	body := parseJSON(t, w)
	generated := body["data"].(map[string]any)
	assert.Equal(t, float64(1), generated["generated_count"])
	assert.Equal(t, float64(0), generated["skipped_count"])
	invoices := generated["invoices"].([]any)
	require.Len(t, invoices, 1)
	invoice := invoices[0].(map[string]any)
	invoiceID, err := uuid.Parse(invoice["id"].(string))
	require.NoError(t, err)
	assert.Equal(t, float64(125000), invoice["subtotal_amount_paise"])
	assert.Equal(t, float64(125000), invoice["total_amount_paise"])
	assert.Equal(t, float64(125000), invoice["balance_amount_paise"])
	assert.Equal(t, true, invoice["allow_partial_payment"])
	assert.Equal(t, float64(50000), invoice["minimum_partial_amount_paise"])

	w = doRequest(suite.Server, http.MethodPost, "/api/v1/admin/invoices/generate", map[string]any{
		"assignment_id":        assignmentID.String(),
		"issue_date":           "2026-06-01",
		"due_date":             "2026-06-10",
		"billing_period_start": "2026-06-01",
		"billing_period_end":   "2026-06-30",
	}, bearer(tenantAccessToken))
	require.Equal(t, http.StatusCreated, w.Code)
	body = parseJSON(t, w)
	generated = body["data"].(map[string]any)
	assert.Equal(t, float64(0), generated["generated_count"])
	assert.Equal(t, float64(1), generated["skipped_count"])

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/invoices?student_id="+studentID.String(), nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	data := body["data"].([]any)
	require.Len(t, data, 1)
	assert.Equal(t, invoiceID.String(), data[0].(map[string]any)["id"])

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/invoices/"+invoiceID.String(), nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	invoice = body["data"].(map[string]any)
	items := invoice["items"].([]any)
	require.Len(t, items, 2)
	assert.Equal(t, float64(100000), items[0].(map[string]any)["amount_paise"])
	assert.Equal(t, float64(25000), items[1].(map[string]any)["amount_paise"])

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/students/"+studentID.String()+"/ledger", nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	ledger := body["data"].(map[string]any)
	assert.Equal(t, float64(125000), ledger["opening_balance_paise"])
	assert.Equal(t, float64(125000), ledger["total_billed_paise"])
	assert.Equal(t, float64(250000), ledger["balance_paise"])
	entries := ledger["entries"].([]any)
	require.Len(t, entries, 2)

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/parent/children/"+studentID.String()+"/dues", nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	dues := body["data"].(map[string]any)
	assert.Equal(t, float64(125000), dues["total_due_paise"])
	assert.Equal(t, true, dues["allow_partial"])
	assert.Equal(t, float64(50000), dues["minimum_payable_paise"])

	var auditCount int
	err = suite.Pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND action IN ('fee_structure.created', 'fee_assignment.created', 'invoices.generated')`, tenantID).Scan(&auditCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, auditCount, 3)

	otherToken, _ := createTenantSession(t, suite, adminID, "Other Billing School", "other-billing")
	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/invoices/"+invoiceID.String(), nil, bearer(otherToken))
	require.Equal(t, http.StatusNotFound, w.Code)
}

func createFeeHead(t *testing.T, suite *TestSuite, token, name, code, category string) uuid.UUID {
	t.Helper()
	w := doRequest(suite.Server, http.MethodPost, "/api/v1/admin/fee-heads", map[string]any{
		"name":     name,
		"code":     code,
		"category": category,
		"status":   "active",
	}, bearer(token))
	require.Equal(t, http.StatusCreated, w.Code)
	return responseID(t, w)
}

func createFeeStructure(t *testing.T, suite *TestSuite, token string, academicYearID, tuitionHeadID, examHeadID uuid.UUID) uuid.UUID {
	t.Helper()
	w := doRequest(suite.Server, http.MethodPost, "/api/v1/admin/fee-structures", map[string]any{
		"academic_year_id":             academicYearID.String(),
		"name":                         "Class 10 Monthly Fee",
		"code":                         "C10-MONTHLY",
		"billing_cycle":                "monthly",
		"status":                       "active",
		"allow_partial_payment":        true,
		"minimum_partial_amount_paise": 50000,
		"due_day":                      10,
		"items": []map[string]any{
			{
				"fee_head_id":  tuitionHeadID.String(),
				"name":         "Tuition",
				"amount_paise": 100000,
				"sort_order":   1,
			},
			{
				"fee_head_id":  examHeadID.String(),
				"name":         "Exam",
				"amount_paise": 25000,
				"sort_order":   2,
			},
		},
	}, bearer(token))
	require.Equal(t, http.StatusCreated, w.Code)
	body := parseJSON(t, w)
	data := body["data"].(map[string]any)
	items := data["items"].([]any)
	require.Len(t, items, 2)
	assert.Equal(t, float64(125000), items[0].(map[string]any)["amount_paise"].(float64)+items[1].(map[string]any)["amount_paise"].(float64))
	return responseID(t, w)
}

func createFeeAssignment(t *testing.T, suite *TestSuite, token string, feeStructureID, classID uuid.UUID) uuid.UUID {
	t.Helper()
	w := doRequest(suite.Server, http.MethodPost, "/api/v1/admin/fee-assignments", map[string]any{
		"fee_structure_id": feeStructureID.String(),
		"assignment_type":  "class",
		"class_id":         classID.String(),
		"effective_from":   "2026-04-01",
	}, bearer(token))
	require.Equal(t, http.StatusCreated, w.Code)
	return responseID(t, w)
}
