package e2e

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemindersReportsExports_E2E(t *testing.T) {
	suite := SetupSuite(t)
	truncateAndReseed(t, suite.Pool, suite.Redis)

	adminID := SeedUser(t, suite.Pool, suite.Hasher, "academic-admin@example.com", "password123", []string{"super_admin"})
	tenantAccessToken, tenantID := createTenantSession(t, suite, adminID, "Reports School", "reports-school")

	academicYearID := createAcademicYear(t, suite, tenantAccessToken)
	classID := createClass(t, suite, tenantAccessToken)
	sectionID := createSection(t, suite, tenantAccessToken, academicYearID, classID)
	guardianID := createGuardian(t, suite, tenantAccessToken)
	studentID := createStudent(t, suite, tenantAccessToken, academicYearID, classID, sectionID, guardianID)

	tuitionHeadID := createFeeHead(t, suite, tenantAccessToken, "Tuition Fee", "TUITION", "tuition")
	examHeadID := createFeeHead(t, suite, tenantAccessToken, "Exam Fee", "EXAM", "exam")
	feeStructureID := createFeeStructure(t, suite, tenantAccessToken, academicYearID, tuitionHeadID, examHeadID)
	assignmentID := createFeeAssignment(t, suite, tenantAccessToken, feeStructureID, classID)

	juneInvoiceID := generateInvoice(t, suite, tenantAccessToken, assignmentID, "2026-06-01", "2026-06-10", "2026-06-01", "2026-06-30")
	julyInvoiceID := generateInvoice(t, suite, tenantAccessToken, assignmentID, "2026-07-01", "2026-07-10", "2026-07-01", "2026-07-31")
	w := doRequest(suite.Server, http.MethodPost, "/api/v1/admin/offline-payments", map[string]any{
		"student_id":       studentID.String(),
		"payment_method":   "cash",
		"received_on":      "2026-07-05",
		"reference_number": "CASH-OPS-001",
		"allocations": []map[string]any{
			{"invoice_id": julyInvoiceID.String(), "amount_paise": 125000},
		},
	}, bearer(tenantAccessToken))
	require.Equal(t, http.StatusCreated, w.Code)

	w = doRequest(suite.Server, http.MethodPost, "/api/v1/admin/reminder-templates", map[string]any{
		"name":    "Polite Overdue Email",
		"code":    "OVERDUE_EMAIL",
		"channel": "email",
		"subject": "Fee reminder for {{student_name}}",
		"body":    "Invoice {{invoice_number}} has {{amount_due}} due on {{due_date}}.",
		"tone":    "polite",
		"status":  "active",
	}, bearer(tenantAccessToken))
	require.Equal(t, http.StatusCreated, w.Code)
	templateID := responseID(t, w)

	w = doRequest(suite.Server, http.MethodPost, "/api/v1/admin/reminder-rules", map[string]any{
		"name":            "Manual overdue email",
		"code":            "MANUAL_OVERDUE_EMAIL",
		"template_id":     templateID.String(),
		"channel":         "email",
		"trigger_type":    "manual",
		"target_statuses": []string{"issued", "partially_paid", "overdue"},
		"max_attempts":    2,
	}, bearer(tenantAccessToken))
	require.Equal(t, http.StatusCreated, w.Code)
	ruleID := responseID(t, w)

	processNow := true
	w = doRequest(suite.Server, http.MethodPost, "/api/v1/admin/reminders/send", map[string]any{
		"rule_id":     ruleID.String(),
		"invoice_ids": []string{juneInvoiceID.String()},
		"process_now": processNow,
	}, bearer(tenantAccessToken))
	require.Equalf(t, http.StatusCreated, w.Code, "body: %s", w.Body.String())
	body := parseJSON(t, w)
	sendData := body["data"].(map[string]any)
	assert.Equal(t, float64(1), sendData["skipped_count"])
	logs := sendData["reminder_logs"].([]any)
	require.Len(t, logs, 1)
	log := logs[0].(map[string]any)
	assert.Equal(t, "email", log["channel"])
	assert.Equal(t, "riya.sharma@example.test", log["recipient"])
	assert.Equal(t, juneInvoiceID.String(), log["invoice_id"])
	assert.Equal(t, "skipped", log["status"])

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/reminder-logs?invoice_id="+juneInvoiceID.String(), nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	reminderLogs := body["data"].([]any)
	require.Len(t, reminderLogs, 1)
	assert.Equal(t, "skipped", reminderLogs[0].(map[string]any)["status"])

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/dashboard?as_of=2026-07-20", nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	dashboard := body["data"].(map[string]any)
	assert.Equal(t, float64(1), dashboard["active_students"])
	assert.Equal(t, float64(125000), dashboard["total_due_paise"])
	assert.Equal(t, float64(125000), dashboard["overdue_paise"])
	assert.Equal(t, float64(1), dashboard["defaulter_count"])

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/reports/collections?from=2026-07-01&to=2026-07-31", nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	collections := body["data"].([]any)
	require.Len(t, collections, 1)
	assert.Equal(t, "cash", collections[0].(map[string]any)["payment_method"])
	assert.Equal(t, float64(125000), collections[0].(map[string]any)["amount_paise"])

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/reports/defaulters?as_of=2026-06-20", nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	defaulters := body["data"].([]any)
	require.Len(t, defaulters, 1)
	defaulter := defaulters[0].(map[string]any)
	assert.Equal(t, "ADM-100", defaulter["admission_number"])
	assert.Equal(t, float64(125000), defaulter["total_due_paise"])
	assert.Equal(t, "skipped", defaulter["last_reminder_status"])

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/reports/dues?as_of=2026-07-20", nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	dues := body["data"].([]any)
	require.Len(t, dues, 1)
	assert.Equal(t, float64(250000), dues[0].(map[string]any)["total_billed_paise"])
	assert.Equal(t, float64(125000), dues[0].(map[string]any)["total_paid_paise"])
	assert.Equal(t, float64(125000), dues[0].(map[string]any)["total_due_paise"])

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/reports/payment-methods?from=2026-07-01&to=2026-07-31", nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	methods := body["data"].([]any)
	require.Len(t, methods, 1)
	assert.Equal(t, "cash", methods[0].(map[string]any)["payment_method"])
	assert.Equal(t, float64(125000), methods[0].(map[string]any)["amount_paise"])

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/reports/fee-heads?from=2026-07-01&to=2026-07-31", nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	feeHeads := body["data"].([]any)
	require.Len(t, feeHeads, 2)

	w = doRequest(suite.Server, http.MethodPost, "/api/v1/admin/exports", map[string]any{
		"export_type": "defaulters",
		"format":      "csv",
		"as_of":       "2026-06-20",
	}, bearer(tenantAccessToken))
	require.Equal(t, http.StatusCreated, w.Code)
	body = parseJSON(t, w)
	exportData := body["data"].(map[string]any)
	assert.Equal(t, "succeeded", exportData["status"])
	assert.Equal(t, float64(1), exportData["row_count"])
	exportID, err := uuid.Parse(exportData["id"].(string))
	require.NoError(t, err)

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/exports/"+exportID.String()+"/download", nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/csv; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "admission_number")
	assert.Contains(t, w.Body.String(), "ADM-100")

	var auditCount int
	err = suite.Pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND action IN ('reminder_template.created', 'reminder_rule.created', 'reminders.queued', 'report.exported')`, tenantID).Scan(&auditCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, auditCount, 4)

	otherToken, _ := createTenantSession(t, suite, adminID, "Other Reports School", "other-reports")
	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/exports/"+exportID.String(), nil, bearer(otherToken))
	require.Equal(t, http.StatusNotFound, w.Code)
}
