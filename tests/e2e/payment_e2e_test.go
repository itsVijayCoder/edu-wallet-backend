package e2e

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPaymentSecret = "test_payment_secret"

func TestPaymentsReceiptsLedger_E2E(t *testing.T) {
	suite := SetupSuite(t)
	truncateAndReseed(t, suite.Pool, suite.Redis)

	adminID := SeedUser(t, suite.Pool, suite.Hasher, "academic-admin@example.com", "password123", []string{"super_admin"})
	tenantAccessToken, tenantID := createTenantSession(t, suite, adminID, "Payments School", "payments-school")

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

	w := doRequest(suite.Server, http.MethodPost, "/api/v1/parent/payments/orders", map[string]any{
		"student_id":      studentID.String(),
		"invoice_ids":     []string{juneInvoiceID.String()},
		"amount_paise":    50000,
		"idempotency_key": "june-partial",
	}, bearer(tenantAccessToken))
	require.Equal(t, http.StatusCreated, w.Code)
	body := parseJSON(t, w)
	orderData := body["data"].(map[string]any)
	orderID := orderData["order_id"].(string)
	assert.Equal(t, float64(50000), orderData["amount_paise"])

	w = doRequest(suite.Server, http.MethodPost, "/api/v1/parent/payments/verify", map[string]any{
		"provider_order_id":   orderID,
		"provider_payment_id": "pay_partial_001",
		"signature":           testSignature(orderID + "|" + "pay_partial_001"),
		"payment_method":      "upi",
	}, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	verifyData := body["data"].(map[string]any)
	firstReceipt := verifyData["receipt"].(map[string]any)
	firstReceiptID, err := uuid.Parse(firstReceipt["id"].(string))
	require.NoError(t, err)
	assert.Equal(t, float64(50000), firstReceipt["amount_paise"])

	w = doRequest(suite.Server, http.MethodPost, "/api/v1/parent/payments/verify", map[string]any{
		"provider_order_id":   orderID,
		"provider_payment_id": "pay_partial_001",
		"signature":           testSignature(orderID + "|" + "pay_partial_001"),
		"payment_method":      "upi",
	}, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	duplicateReceipt := body["data"].(map[string]any)["receipt"].(map[string]any)
	assert.Equal(t, firstReceipt["receipt_number"], duplicateReceipt["receipt_number"])

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/invoices/"+juneInvoiceID.String(), nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	juneInvoice := body["data"].(map[string]any)
	assert.Equal(t, "partially_paid", juneInvoice["status"])
	assert.Equal(t, float64(50000), juneInvoice["paid_amount_paise"])
	assert.Equal(t, float64(75000), juneInvoice["balance_amount_paise"])

	w = doRequest(suite.Server, http.MethodPost, "/api/v1/parent/payments/orders", map[string]any{
		"student_id":  studentID.String(),
		"invoice_ids": []string{juneInvoiceID.String()},
	}, bearer(tenantAccessToken))
	require.Equal(t, http.StatusCreated, w.Code)
	body = parseJSON(t, w)
	webhookOrderID := body["data"].(map[string]any)["order_id"].(string)
	webhookPayload := webhookPaymentPayload(t, webhookOrderID, "pay_webhook_001", 75000, "upi", "payment.captured", "captured")
	w = doRawRequest(suite.Server, http.MethodPost, "/api/v1/webhooks/razorpay", webhookPayload, map[string]string{
		"Content-Type":         "application/json",
		"X-Razorpay-Signature": testSignatureBytes(webhookPayload),
		"X-Razorpay-Event-Id":  "evt_payment_captured_001",
	})
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	assert.Equal(t, "processed", body["data"].(map[string]any)["status"])

	w = doRawRequest(suite.Server, http.MethodPost, "/api/v1/webhooks/razorpay", webhookPayload, map[string]string{
		"Content-Type":         "application/json",
		"X-Razorpay-Signature": testSignatureBytes(webhookPayload),
		"X-Razorpay-Event-Id":  "evt_payment_captured_001",
	})
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	assert.Equal(t, "duplicate", body["data"].(map[string]any)["status"])

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/invoices/"+juneInvoiceID.String(), nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	juneInvoice = body["data"].(map[string]any)
	assert.Equal(t, "paid", juneInvoice["status"])
	assert.Equal(t, float64(125000), juneInvoice["paid_amount_paise"])
	assert.Equal(t, float64(0), juneInvoice["balance_amount_paise"])

	julyInvoiceID := generateInvoice(t, suite, tenantAccessToken, assignmentID, "2026-07-01", "2026-07-10", "2026-07-01", "2026-07-31")
	w = doRequest(suite.Server, http.MethodPost, "/api/v1/admin/offline-payments", map[string]any{
		"student_id":       studentID.String(),
		"payment_method":   "cash",
		"received_on":      "2026-07-05",
		"reference_number": "CASH-001",
		"allocations": []map[string]any{
			{"invoice_id": julyInvoiceID.String(), "amount_paise": 125000},
		},
	}, bearer(tenantAccessToken))
	require.Equal(t, http.StatusCreated, w.Code)
	body = parseJSON(t, w)
	offlineReceipt := body["data"].(map[string]any)["receipt"].(map[string]any)
	assert.Equal(t, float64(125000), offlineReceipt["amount_paise"])
	assert.Equal(t, "cash", offlineReceipt["payment_method"])

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/receipts?student_id="+studentID.String(), nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	receipts := body["data"].([]any)
	require.Len(t, receipts, 3)

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/receipts/"+firstReceiptID.String()+"/download", nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "%PDF-1.4")

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/students/"+studentID.String()+"/ledger", nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	ledger := body["data"].(map[string]any)
	assert.Equal(t, float64(250000), ledger["total_billed_paise"])
	assert.Equal(t, float64(250000), ledger["total_paid_paise"])
	assert.Equal(t, float64(125000), ledger["balance_paise"])
	entries := ledger["entries"].([]any)
	require.Len(t, entries, 6)

	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/payment-events?student_id="+studentID.String(), nil, bearer(tenantAccessToken))
	require.Equal(t, http.StatusOK, w.Code)
	body = parseJSON(t, w)
	events := body["data"].([]any)
	assert.GreaterOrEqual(t, len(events), 3)

	var auditCount int
	err = suite.Pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND action IN ('payment.received', 'payment.offline_recorded', 'receipt.issued')`, tenantID).Scan(&auditCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, auditCount, 5)

	otherToken, _ := createTenantSession(t, suite, adminID, "Other Payments School", "other-payments")
	w = doRequest(suite.Server, http.MethodGet, "/api/v1/admin/receipts/"+firstReceiptID.String(), nil, bearer(otherToken))
	require.Equal(t, http.StatusNotFound, w.Code)
}

func generateInvoice(t *testing.T, suite *TestSuite, token string, assignmentID uuid.UUID, issueDate, dueDate, periodStart, periodEnd string) uuid.UUID {
	t.Helper()
	w := doRequest(suite.Server, http.MethodPost, "/api/v1/admin/invoices/generate", map[string]any{
		"assignment_id":        assignmentID.String(),
		"issue_date":           issueDate,
		"due_date":             dueDate,
		"billing_period_start": periodStart,
		"billing_period_end":   periodEnd,
	}, bearer(token))
	require.Equal(t, http.StatusCreated, w.Code)
	body := parseJSON(t, w)
	invoices := body["data"].(map[string]any)["invoices"].([]any)
	require.Len(t, invoices, 1)
	id, err := uuid.Parse(invoices[0].(map[string]any)["id"].(string))
	require.NoError(t, err)
	return id
}

func webhookPaymentPayload(t *testing.T, orderID, paymentID string, amount int64, method, event, status string) []byte {
	t.Helper()
	payload := map[string]any{
		"event": event,
		"payload": map[string]any{
			"payment": map[string]any{
				"entity": map[string]any{
					"id":       paymentID,
					"order_id": orderID,
					"amount":   amount,
					"method":   method,
					"status":   status,
					"captured": status == "captured",
				},
			},
		},
	}
	b, err := json.Marshal(payload)
	require.NoError(t, err)
	return b
}

func doRawRequest(server *gin.Engine, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

func testSignature(message string) string {
	return testSignatureBytes([]byte(message))
}

func testSignatureBytes(message []byte) string {
	mac := hmac.New(sha256.New, []byte(testPaymentSecret))
	_, _ = mac.Write(message)
	return hex.EncodeToString(mac.Sum(nil))
}
