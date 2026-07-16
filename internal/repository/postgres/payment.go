package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository"
)

var _ repository.PaymentRepository = (*paymentRepository)(nil)

type paymentRepository struct {
	db database.DBTX
}

func NewPaymentRepository(db database.DBTX) repository.PaymentRepository {
	return &paymentRepository{db: db}
}

var (
	allowedPaymentSortColumns      = map[string]bool{"created_at": true, "paid_at": true, "amount_paise": true, "status": true, "payment_method": true}
	allowedReceiptSortColumns      = map[string]bool{"created_at": true, "issue_date": true, "receipt_number": true, "amount_paise": true, "status": true}
	allowedPaymentEventSortColumns = map[string]bool{"occurred_at": true, "created_at": true, "event_type": true, "status": true, "amount_paise": true}
)

func (r *paymentRepository) CreatePaymentAttempt(ctx context.Context, attempt *model.PaymentAttempt) error {
	const query = `INSERT INTO payment_attempts (
			tenant_id, student_id, provider, provider_order_id, idempotency_key, status,
			amount_paise, currency, checkout_url, expires_at, created_by, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		attempt.TenantID,
		attempt.StudentID,
		attempt.Provider,
		attempt.ProviderOrderID,
		attempt.IdempotencyKey,
		attempt.Status,
		attempt.AmountPaise,
		attempt.Currency,
		attempt.CheckoutURL,
		attempt.ExpiresAt,
		attempt.CreatedBy,
		mustJSON(attempt.Metadata),
	).Scan(&attempt.ID, &attempt.CreatedAt, &attempt.UpdatedAt)
}

func (r *paymentRepository) GetPaymentAttempt(ctx context.Context, tenantID, id uuid.UUID) (*model.PaymentAttempt, error) {
	const query = paymentAttemptSelect + ` WHERE pa.tenant_id = $1 AND pa.id = $2`
	item, err := r.scanPaymentAttempt(ctx, query, tenantID, id)
	if err != nil || item == nil {
		return item, err
	}
	allocations, err := r.ListPaymentAttemptAllocations(ctx, tenantID, item.ID)
	if err != nil {
		return nil, err
	}
	item.Allocations = allocations
	return item, nil
}

func (r *paymentRepository) GetPaymentAttemptByProviderOrderID(ctx context.Context, tenantID uuid.UUID, provider, providerOrderID string) (*model.PaymentAttempt, error) {
	const query = paymentAttemptSelect + ` WHERE pa.tenant_id = $1 AND pa.provider = $2 AND pa.provider_order_id = $3`
	item, err := r.scanPaymentAttempt(ctx, query, tenantID, provider, providerOrderID)
	if err != nil || item == nil {
		return item, err
	}
	allocations, err := r.ListPaymentAttemptAllocations(ctx, tenantID, item.ID)
	if err != nil {
		return nil, err
	}
	item.Allocations = allocations
	return item, nil
}

func (r *paymentRepository) GetPaymentAttemptByProviderOrderIDAnyTenant(ctx context.Context, provider, providerOrderID string) (*model.PaymentAttempt, error) {
	const query = paymentAttemptSelect + ` WHERE pa.provider = $1 AND pa.provider_order_id = $2`
	item, err := r.scanPaymentAttempt(ctx, query, provider, providerOrderID)
	if err != nil || item == nil {
		return item, err
	}
	allocations, err := r.ListPaymentAttemptAllocations(ctx, item.TenantID, item.ID)
	if err != nil {
		return nil, err
	}
	item.Allocations = allocations
	return item, nil
}

func (r *paymentRepository) GetPaymentAttemptByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, idempotencyKey string) (*model.PaymentAttempt, error) {
	const query = paymentAttemptSelect + ` WHERE pa.tenant_id = $1 AND pa.idempotency_key = $2`
	item, err := r.scanPaymentAttempt(ctx, query, tenantID, idempotencyKey)
	if err != nil || item == nil {
		return item, err
	}
	allocations, err := r.ListPaymentAttemptAllocations(ctx, tenantID, item.ID)
	if err != nil {
		return nil, err
	}
	item.Allocations = allocations
	return item, nil
}

func (r *paymentRepository) UpdatePaymentAttemptProviderOrder(ctx context.Context, tenantID, id uuid.UUID, providerOrderID, checkoutURL, status string, metadata map[string]any) error {
	const query = `UPDATE payment_attempts
		SET provider_order_id = $1,
			checkout_url = $2,
			status = $3,
			metadata = metadata || $4::jsonb,
			updated_at = NOW()
		WHERE tenant_id = $5 AND id = $6`
	_, err := r.db.Exec(ctx, query, providerOrderID, checkoutURL, status, mustJSON(metadata), tenantID, id)
	return err
}

func (r *paymentRepository) UpdatePaymentAttemptStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	_, err := r.db.Exec(ctx, `UPDATE payment_attempts SET status = $1, updated_at = NOW() WHERE tenant_id = $2 AND id = $3`, status, tenantID, id)
	return err
}

func (r *paymentRepository) CreatePaymentAttemptAllocations(ctx context.Context, allocations []model.PaymentAllocation) error {
	const query = `INSERT INTO payment_attempt_allocations (
			tenant_id, attempt_id, invoice_id, amount_paise
		) VALUES ($1, $2, $3, $4)
		RETURNING created_at`

	for i := range allocations {
		if err := r.db.QueryRow(ctx, query,
			allocations[i].TenantID,
			allocations[i].AttemptID,
			allocations[i].InvoiceID,
			allocations[i].AmountPaise,
		).Scan(&allocations[i].CreatedAt); err != nil {
			return err
		}
	}
	return nil
}

func (r *paymentRepository) ListPaymentAttemptAllocations(ctx context.Context, tenantID, attemptID uuid.UUID) ([]model.PaymentAllocation, error) {
	query := paymentAttemptAllocationSelect + ` WHERE paa.tenant_id = $1 AND paa.attempt_id = $2 ORDER BY i.due_date ASC, i.created_at ASC`
	rows, err := r.db.Query(ctx, query, tenantID, attemptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.PaymentAllocation{}
	for rows.Next() {
		item, err := scanPaymentAttemptAllocationScanner(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *paymentRepository) GetInvoiceForPayment(ctx context.Context, tenantID, invoiceID uuid.UUID) (*model.Invoice, error) {
	query := invoiceSelect + ` WHERE i.tenant_id = $1 AND i.id = $2 FOR UPDATE OF i`
	return r.scanInvoiceForPayment(ctx, query, tenantID, invoiceID)
}

func (r *paymentRepository) ApplyInvoicePayment(ctx context.Context, tenantID, invoiceID uuid.UUID, amountPaise int64, asOf time.Time) (*model.Invoice, error) {
	query := `WITH updated AS (
		UPDATE invoices i
		SET paid_amount_paise = i.paid_amount_paise + $3,
			balance_amount_paise = i.total_amount_paise - (i.paid_amount_paise + $3),
			status = CASE
				WHEN i.total_amount_paise - (i.paid_amount_paise + $3) = 0 THEN 'paid'
				WHEN i.paid_amount_paise + $3 > 0 THEN 'partially_paid'
				WHEN i.due_date < $4 THEN 'overdue'
				ELSE 'issued'
			END,
			updated_at = NOW()
		WHERE i.tenant_id = $1
		  AND i.id = $2
		  AND i.status NOT IN ('cancelled', 'void')
		  AND i.balance_amount_paise >= $3
		RETURNING i.id
	)
	` + invoiceSelect + ` JOIN updated u ON u.id = i.id`
	return r.scanInvoiceForPayment(ctx, query, tenantID, invoiceID, amountPaise, asOf)
}

func (r *paymentRepository) CreatePayment(ctx context.Context, payment *model.Payment) error {
	const query = `INSERT INTO payments (
			tenant_id, attempt_id, student_id, provider, payment_method, status,
			amount_paise, amount_applied_paise, currency, gateway_order_id,
			gateway_payment_id, gateway_signature, external_reference, paid_at,
			verified_at, received_by, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15, $16, $17
		)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		payment.TenantID,
		payment.AttemptID,
		payment.StudentID,
		payment.Provider,
		payment.PaymentMethod,
		payment.Status,
		payment.AmountPaise,
		payment.AmountAppliedPaise,
		payment.Currency,
		payment.GatewayOrderID,
		payment.GatewayPaymentID,
		payment.GatewaySignature,
		payment.ExternalReference,
		payment.PaidAt,
		payment.VerifiedAt,
		payment.ReceivedBy,
		mustJSON(payment.Metadata),
	).Scan(&payment.ID, &payment.CreatedAt, &payment.UpdatedAt)
}

func (r *paymentRepository) GetPayment(ctx context.Context, tenantID, id uuid.UUID) (*model.Payment, error) {
	const query = paymentSelect + ` WHERE p.tenant_id = $1 AND p.id = $2`
	item, err := r.scanPayment(ctx, query, tenantID, id)
	if err != nil || item == nil {
		return item, err
	}
	if err := r.withPaymentChildren(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *paymentRepository) GetPaymentByGatewayPaymentID(ctx context.Context, tenantID uuid.UUID, provider, gatewayPaymentID string) (*model.Payment, error) {
	const query = paymentSelect + ` WHERE p.tenant_id = $1 AND p.provider = $2 AND p.gateway_payment_id = $3`
	item, err := r.scanPayment(ctx, query, tenantID, provider, gatewayPaymentID)
	if err != nil || item == nil {
		return item, err
	}
	if err := r.withPaymentChildren(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *paymentRepository) ListPayments(ctx context.Context, tenantID uuid.UUID, filter model.PaymentFilter, params model.PaginationParams) (*model.PaginatedResult[model.Payment], error) {
	params.Normalize()
	where, args := paymentWhere(tenantID, filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM payments p JOIN students s ON s.tenant_id = p.tenant_id AND s.id = p.student_id `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())

	sortCol, sortDir := sanitizeSort(params, allowedPaymentSortColumns, "created_at", "DESC")
	query := fmt.Sprintf(`%s %s ORDER BY p.%s %s LIMIT %s OFFSET %s`, paymentSelect, where, sortCol, sortDir, limitParam, offsetParam)
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Payment, 0, params.PageSize)
	for rows.Next() {
		item, err := scanPaymentScanner(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return model.NewPaginatedResult(items, total, params.Page, params.PageSize), nil
}

func (r *paymentRepository) CreatePaymentAllocations(ctx context.Context, allocations []model.PaymentAllocation) error {
	const query = `INSERT INTO payment_allocations (
			tenant_id, payment_id, invoice_id, amount_paise
		) VALUES ($1, $2, $3, $4)
		RETURNING created_at`

	for i := range allocations {
		if err := r.db.QueryRow(ctx, query,
			allocations[i].TenantID,
			allocations[i].PaymentID,
			allocations[i].InvoiceID,
			allocations[i].AmountPaise,
		).Scan(&allocations[i].CreatedAt); err != nil {
			return err
		}
	}
	return nil
}

func (r *paymentRepository) ListPaymentAllocations(ctx context.Context, tenantID, paymentID uuid.UUID) ([]model.PaymentAllocation, error) {
	query := paymentAllocationSelect + ` WHERE pa.tenant_id = $1 AND pa.payment_id = $2 ORDER BY i.due_date ASC, i.created_at ASC`
	rows, err := r.db.Query(ctx, query, tenantID, paymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.PaymentAllocation{}
	for rows.Next() {
		item, err := scanPaymentAllocationScanner(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *paymentRepository) ListStudentPayments(ctx context.Context, tenantID, studentID uuid.UUID) ([]model.Payment, error) {
	const query = paymentSelect + ` WHERE p.tenant_id = $1 AND p.student_id = $2 AND p.status IN ('success', 'manually_verified', 'settled') ORDER BY COALESCE(p.paid_at, p.created_at) ASC, p.created_at ASC`
	rows, err := r.db.Query(ctx, query, tenantID, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.Payment{}
	for rows.Next() {
		item, err := scanPaymentScanner(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *paymentRepository) CreateGatewayWebhook(ctx context.Context, webhook *model.GatewayWebhook) error {
	const query = `INSERT INTO gateway_webhooks (
			tenant_id, provider, event_id, event_type, processing_status,
			payload, signature, error_message, processed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, received_at`

	return r.db.QueryRow(ctx, query,
		webhook.TenantID,
		webhook.Provider,
		webhook.EventID,
		webhook.EventType,
		webhook.ProcessingStatus,
		mustJSON(webhook.Payload),
		webhook.Signature,
		webhook.ErrorMessage,
		webhook.ProcessedAt,
	).Scan(&webhook.ID, &webhook.ReceivedAt)
}

func (r *paymentRepository) GetGatewayWebhookByEventID(ctx context.Context, provider, eventID string) (*model.GatewayWebhook, error) {
	const query = gatewayWebhookSelect + ` WHERE provider = $1 AND event_id = $2`
	item, err := scanGatewayWebhookScanner(r.db.QueryRow(ctx, query, provider, eventID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (r *paymentRepository) UpdateGatewayWebhookStatus(ctx context.Context, tenantID, id uuid.UUID, status, errorMessage string) error {
	processedAt := any(nil)
	if status == "processed" || status == "failed" || status == "ignored" || status == "duplicate" {
		processedAt = time.Now().UTC()
	}
	_, err := r.db.Exec(ctx, `UPDATE gateway_webhooks
		SET processing_status = $1, error_message = $2, processed_at = COALESCE($3::timestamptz, processed_at)
		WHERE tenant_id = $4 AND id = $5`, status, errorMessage, processedAt, tenantID, id)
	return err
}

func (r *paymentRepository) NextReceiptNumber(ctx context.Context, tenantID, academicYearID uuid.UUID, branchID *uuid.UUID, prefix string) (int64, error) {
	scopeKey := receiptSeriesScopeKey(academicYearID, branchID, prefix)
	const query = `INSERT INTO receipt_series (
			tenant_id, academic_year_id, branch_id, prefix, scope_key, next_number
		) VALUES ($1, $2, $3, $4, $5, 2)
		ON CONFLICT (tenant_id, scope_key)
		DO UPDATE SET next_number = receipt_series.next_number + 1,
			updated_at = NOW()
		RETURNING next_number - 1`

	var sequence int64
	if err := r.db.QueryRow(ctx, query, tenantID, academicYearID, branchID, prefix, scopeKey).Scan(&sequence); err != nil {
		return 0, err
	}
	return sequence, nil
}

func (r *paymentRepository) CreateReceipt(ctx context.Context, receipt *model.Receipt) error {
	const query = `INSERT INTO receipts (
			tenant_id, receipt_number, payment_id, student_id, academic_year_id,
			branch_id, status, issue_date, currency, amount_paise, payment_method,
			issued_by, issued_at, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		receipt.TenantID,
		receipt.ReceiptNumber,
		receipt.PaymentID,
		receipt.StudentID,
		receipt.AcademicYearID,
		receipt.BranchID,
		receipt.Status,
		receipt.IssueDate,
		receipt.Currency,
		receipt.AmountPaise,
		receipt.PaymentMethod,
		receipt.IssuedBy,
		receipt.IssuedAt,
		mustJSON(receipt.Metadata),
	).Scan(&receipt.ID, &receipt.CreatedAt, &receipt.UpdatedAt)
}

func (r *paymentRepository) GetReceipt(ctx context.Context, tenantID, id uuid.UUID) (*model.Receipt, error) {
	const query = receiptSelect + ` WHERE r.tenant_id = $1 AND r.id = $2`
	item, err := r.scanReceipt(ctx, query, tenantID, id)
	if err != nil || item == nil {
		return item, err
	}
	if err := r.withReceiptChildren(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *paymentRepository) GetReceiptByPaymentID(ctx context.Context, tenantID, paymentID uuid.UUID) (*model.Receipt, error) {
	const query = receiptSelect + ` WHERE r.tenant_id = $1 AND r.payment_id = $2`
	item, err := r.scanReceipt(ctx, query, tenantID, paymentID)
	if err != nil || item == nil {
		return item, err
	}
	if err := r.withReceiptChildren(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *paymentRepository) ListReceipts(ctx context.Context, tenantID uuid.UUID, filter model.ReceiptFilter, params model.PaginationParams) (*model.PaginatedResult[model.Receipt], error) {
	params.Normalize()
	where, args := receiptWhere(tenantID, filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM receipts r JOIN students s ON s.tenant_id = r.tenant_id AND s.id = r.student_id `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())

	sortCol, sortDir := sanitizeSort(params, allowedReceiptSortColumns, "created_at", "DESC")
	query := fmt.Sprintf(`%s %s ORDER BY r.%s %s LIMIT %s OFFSET %s`, receiptSelect, where, sortCol, sortDir, limitParam, offsetParam)
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Receipt, 0, params.PageSize)
	for rows.Next() {
		item, err := scanReceiptScanner(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return model.NewPaginatedResult(items, total, params.Page, params.PageSize), nil
}

func (r *paymentRepository) ListStudentReceipts(ctx context.Context, tenantID, studentID uuid.UUID) ([]model.Receipt, error) {
	const query = receiptSelect + ` WHERE r.tenant_id = $1 AND r.student_id = $2 ORDER BY r.issue_date ASC, r.created_at ASC`
	rows, err := r.db.Query(ctx, query, tenantID, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.Receipt{}
	for rows.Next() {
		item, err := scanReceiptScanner(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *paymentRepository) CreateOfflinePaymentReference(ctx context.Context, ref *model.OfflinePaymentReference) error {
	const query = `INSERT INTO offline_payment_references (
			tenant_id, payment_id, payment_method, reference_number, bank_name,
			instrument_date, deposited_at, clearance_status, remarks, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		ref.TenantID,
		ref.PaymentID,
		ref.PaymentMethod,
		ref.ReferenceNumber,
		ref.BankName,
		ref.InstrumentDate,
		ref.DepositedAt,
		ref.ClearanceStatus,
		ref.Remarks,
		mustJSON(ref.Metadata),
	).Scan(&ref.ID, &ref.CreatedAt, &ref.UpdatedAt)
}

func (r *paymentRepository) GetOfflinePaymentReferenceByPaymentID(ctx context.Context, tenantID, paymentID uuid.UUID) (*model.OfflinePaymentReference, error) {
	const query = `SELECT id, tenant_id, payment_id, payment_method, reference_number,
			bank_name, instrument_date, deposited_at, clearance_status, remarks,
			metadata, created_at, updated_at
		FROM offline_payment_references
		WHERE tenant_id = $1 AND payment_id = $2`
	item, err := scanOfflinePaymentReferenceScanner(r.db.QueryRow(ctx, query, tenantID, paymentID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (r *paymentRepository) CreatePaymentEvent(ctx context.Context, event *model.PaymentEvent) error {
	const query = `INSERT INTO payment_events (
			tenant_id, payment_id, attempt_id, receipt_id, student_id, event_type,
			status, amount_paise, message, metadata, occurred_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, COALESCE($11::timestamptz, NOW()))
		RETURNING id, occurred_at, created_at`
	var occurredAt any
	if !event.OccurredAt.IsZero() {
		occurredAt = event.OccurredAt
	}

	return r.db.QueryRow(ctx, query,
		event.TenantID,
		event.PaymentID,
		event.AttemptID,
		event.ReceiptID,
		event.StudentID,
		event.EventType,
		event.Status,
		event.AmountPaise,
		event.Message,
		mustJSON(event.Metadata),
		occurredAt,
	).Scan(&event.ID, &event.OccurredAt, &event.CreatedAt)
}

func (r *paymentRepository) ListPaymentEvents(ctx context.Context, tenantID uuid.UUID, filter model.PaymentEventFilter, params model.PaginationParams) (*model.PaginatedResult[model.PaymentEvent], error) {
	params.Normalize()
	where, args := paymentEventWhere(tenantID, filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM payment_events pe `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())

	sortCol, sortDir := sanitizeSort(params, allowedPaymentEventSortColumns, "occurred_at", "DESC")
	query := fmt.Sprintf(`%s %s ORDER BY pe.%s %s LIMIT %s OFFSET %s`, paymentEventSelect, where, sortCol, sortDir, limitParam, offsetParam)
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.PaymentEvent, 0, params.PageSize)
	for rows.Next() {
		item, err := scanPaymentEventScanner(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return model.NewPaginatedResult(items, total, params.Page, params.PageSize), nil
}

func (r *paymentRepository) withPaymentChildren(ctx context.Context, payment *model.Payment) error {
	allocations, err := r.ListPaymentAllocations(ctx, payment.TenantID, payment.ID)
	if err != nil {
		return err
	}
	payment.Allocations = allocations
	receipt, err := r.GetReceiptByPaymentID(ctx, payment.TenantID, payment.ID)
	if err != nil {
		return err
	}
	payment.Receipt = receipt
	return nil
}

func (r *paymentRepository) withReceiptChildren(ctx context.Context, receipt *model.Receipt) error {
	payment, err := r.scanPayment(ctx, paymentSelect+` WHERE p.tenant_id = $1 AND p.id = $2`, receipt.TenantID, receipt.PaymentID)
	if err != nil {
		return err
	}
	receipt.Payment = payment
	if payment != nil {
		allocations, err := r.ListPaymentAllocations(ctx, receipt.TenantID, payment.ID)
		if err != nil {
			return err
		}
		receipt.Allocations = allocations
	}
	return nil
}

const paymentAttemptSelect = `SELECT
		pa.id, pa.tenant_id, pa.student_id, pa.provider, pa.provider_order_id,
		pa.idempotency_key, pa.status, pa.amount_paise, pa.currency, pa.checkout_url,
		pa.expires_at, pa.provider_retry_count, pa.provider_last_error,
		pa.reconciliation_status, pa.reconciled_at, pa.settlement_reference,
		pa.settled_at, pa.created_by, pa.metadata, pa.created_at, pa.updated_at,
		s.id, s.admission_number, s.first_name, s.last_name, s.status
	FROM payment_attempts pa
	JOIN students s ON s.tenant_id = pa.tenant_id AND s.id = pa.student_id`

const paymentSelect = `SELECT
		p.id, p.tenant_id, p.attempt_id, p.student_id, p.provider, p.payment_method,
		p.status, p.amount_paise, p.amount_applied_paise, p.currency, p.gateway_order_id,
		p.gateway_payment_id, p.gateway_signature, p.external_reference, p.paid_at,
		p.verified_at, p.reconciliation_status, p.reconciled_at, p.settlement_reference,
		p.settled_at, p.received_by, p.metadata, p.created_at, p.updated_at,
		s.id, s.admission_number, s.first_name, s.last_name, s.status
	FROM payments p
	JOIN students s ON s.tenant_id = p.tenant_id AND s.id = p.student_id`

const paymentAttemptAllocationSelect = `SELECT
		paa.tenant_id, paa.attempt_id, paa.invoice_id, paa.amount_paise, paa.created_at,
		i.id, i.tenant_id, i.invoice_number, i.student_id, i.academic_year_id, i.class_id,
		i.section_id, i.fee_structure_id, i.assignment_id, i.issue_date, i.due_date,
		i.billing_period_start, i.billing_period_end, i.generation_key, i.status, i.currency,
		i.allow_partial_payment, i.minimum_partial_amount_paise, i.subtotal_amount_paise, i.discount_amount_paise,
		i.fine_amount_paise, i.tax_amount_paise, i.total_amount_paise, i.paid_amount_paise,
		i.balance_amount_paise, i.generated_by, i.metadata, i.issued_at, i.cancelled_at,
		i.voided_at, i.created_at, i.updated_at,
		s.id, s.admission_number, s.first_name, s.last_name, s.status,
		ay.id, ay.name, ay.code,
		c.id, c.name, c.code,
		sec.id, sec.name, sec.code
	FROM payment_attempt_allocations paa
	JOIN invoices i ON i.tenant_id = paa.tenant_id AND i.id = paa.invoice_id
	JOIN students s ON s.tenant_id = i.tenant_id AND s.id = i.student_id
	JOIN academic_years ay ON ay.tenant_id = i.tenant_id AND ay.id = i.academic_year_id
	JOIN classes c ON c.tenant_id = i.tenant_id AND c.id = i.class_id
	JOIN sections sec ON sec.tenant_id = i.tenant_id AND sec.id = i.section_id`

const paymentAllocationSelect = `SELECT
		pa.tenant_id, pa.payment_id, pa.invoice_id, pa.amount_paise, pa.created_at,
		i.id, i.tenant_id, i.invoice_number, i.student_id, i.academic_year_id, i.class_id,
		i.section_id, i.fee_structure_id, i.assignment_id, i.issue_date, i.due_date,
		i.billing_period_start, i.billing_period_end, i.generation_key, i.status, i.currency,
		i.allow_partial_payment, i.minimum_partial_amount_paise, i.subtotal_amount_paise, i.discount_amount_paise,
		i.fine_amount_paise, i.tax_amount_paise, i.total_amount_paise, i.paid_amount_paise,
		i.balance_amount_paise, i.generated_by, i.metadata, i.issued_at, i.cancelled_at,
		i.voided_at, i.created_at, i.updated_at,
		s.id, s.admission_number, s.first_name, s.last_name, s.status,
		ay.id, ay.name, ay.code,
		c.id, c.name, c.code,
		sec.id, sec.name, sec.code
	FROM payment_allocations pa
	JOIN invoices i ON i.tenant_id = pa.tenant_id AND i.id = pa.invoice_id
	JOIN students s ON s.tenant_id = i.tenant_id AND s.id = i.student_id
	JOIN academic_years ay ON ay.tenant_id = i.tenant_id AND ay.id = i.academic_year_id
	JOIN classes c ON c.tenant_id = i.tenant_id AND c.id = i.class_id
	JOIN sections sec ON sec.tenant_id = i.tenant_id AND sec.id = i.section_id`

const gatewayWebhookSelect = `SELECT
		id, tenant_id, provider, event_id, event_type, processing_status,
		payload, signature, error_message, received_at, processed_at
	FROM gateway_webhooks`

const receiptSelect = `SELECT
		r.id, r.tenant_id, r.receipt_number, r.payment_id, r.student_id,
		r.academic_year_id, r.branch_id, r.status, r.issue_date, r.currency,
		r.amount_paise, r.payment_method, r.issued_by, r.issued_at,
		r.cancelled_at, r.metadata, r.created_at, r.updated_at,
		s.id, s.admission_number, s.first_name, s.last_name, s.status,
		ay.id, ay.name, ay.code
	FROM receipts r
	JOIN students s ON s.tenant_id = r.tenant_id AND s.id = r.student_id
	JOIN academic_years ay ON ay.tenant_id = r.tenant_id AND ay.id = r.academic_year_id`

const paymentEventSelect = `SELECT
		pe.id, pe.tenant_id, pe.payment_id, pe.attempt_id, pe.receipt_id, pe.student_id,
		pe.event_type, pe.status, pe.amount_paise, pe.message, pe.metadata,
		pe.occurred_at, pe.created_at,
		s.id, s.admission_number, s.first_name, s.last_name, s.status
	FROM payment_events pe
	JOIN students s ON s.tenant_id = pe.tenant_id AND s.id = pe.student_id`

func (r *paymentRepository) scanPaymentAttempt(ctx context.Context, query string, args ...any) (*model.PaymentAttempt, error) {
	item, err := scanPaymentAttemptScanner(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (r *paymentRepository) scanPayment(ctx context.Context, query string, args ...any) (*model.Payment, error) {
	item, err := scanPaymentScanner(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (r *paymentRepository) scanReceipt(ctx context.Context, query string, args ...any) (*model.Receipt, error) {
	item, err := scanReceiptScanner(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (r *paymentRepository) scanInvoiceForPayment(ctx context.Context, query string, args ...any) (*model.Invoice, error) {
	item, err := scanInvoiceScanner(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func scanPaymentAttemptScanner(row rowScanner) (*model.PaymentAttempt, error) {
	var item model.PaymentAttempt
	var metadata []byte
	student := model.Student{}
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.StudentID,
		&item.Provider,
		&item.ProviderOrderID,
		&item.IdempotencyKey,
		&item.Status,
		&item.AmountPaise,
		&item.Currency,
		&item.CheckoutURL,
		&item.ExpiresAt,
		&item.ProviderRetryCount,
		&item.ProviderLastError,
		&item.ReconciliationStatus,
		&item.ReconciledAt,
		&item.SettlementReference,
		&item.SettledAt,
		&item.CreatedBy,
		&metadata,
		&item.CreatedAt,
		&item.UpdatedAt,
		&student.ID,
		&student.AdmissionNumber,
		&student.FirstName,
		&student.LastName,
		&student.Status,
	); err != nil {
		return nil, err
	}
	item.Metadata = parseJSON(metadata)
	item.Student = &student
	return &item, nil
}

func scanPaymentScanner(row rowScanner) (*model.Payment, error) {
	var item model.Payment
	var metadata []byte
	student := model.Student{}
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.AttemptID,
		&item.StudentID,
		&item.Provider,
		&item.PaymentMethod,
		&item.Status,
		&item.AmountPaise,
		&item.AmountAppliedPaise,
		&item.Currency,
		&item.GatewayOrderID,
		&item.GatewayPaymentID,
		&item.GatewaySignature,
		&item.ExternalReference,
		&item.PaidAt,
		&item.VerifiedAt,
		&item.ReconciliationStatus,
		&item.ReconciledAt,
		&item.SettlementReference,
		&item.SettledAt,
		&item.ReceivedBy,
		&metadata,
		&item.CreatedAt,
		&item.UpdatedAt,
		&student.ID,
		&student.AdmissionNumber,
		&student.FirstName,
		&student.LastName,
		&student.Status,
	); err != nil {
		return nil, err
	}
	item.Metadata = parseJSON(metadata)
	item.Student = &student
	return &item, nil
}

func scanPaymentAttemptAllocationScanner(row rowScanner) (*model.PaymentAllocation, error) {
	var item model.PaymentAllocation
	invoice, err := scanInvoiceTail(row, &item.TenantID, &item.AttemptID, &item.InvoiceID, &item.AmountPaise, &item.CreatedAt)
	if err != nil {
		return nil, err
	}
	item.Invoice = invoice
	return &item, nil
}

func scanPaymentAllocationScanner(row rowScanner) (*model.PaymentAllocation, error) {
	var item model.PaymentAllocation
	invoice, err := scanInvoiceTail(row, &item.TenantID, &item.PaymentID, &item.InvoiceID, &item.AmountPaise, &item.CreatedAt)
	if err != nil {
		return nil, err
	}
	item.Invoice = invoice
	return &item, nil
}

func scanGatewayWebhookScanner(row rowScanner) (*model.GatewayWebhook, error) {
	var item model.GatewayWebhook
	var payload []byte
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Provider,
		&item.EventID,
		&item.EventType,
		&item.ProcessingStatus,
		&payload,
		&item.Signature,
		&item.ErrorMessage,
		&item.ReceivedAt,
		&item.ProcessedAt,
	); err != nil {
		return nil, err
	}
	item.Payload = parseJSON(payload)
	return &item, nil
}

func scanReceiptScanner(row rowScanner) (*model.Receipt, error) {
	var item model.Receipt
	var metadata []byte
	student := model.Student{}
	academicYear := model.AcademicYear{}
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.ReceiptNumber,
		&item.PaymentID,
		&item.StudentID,
		&item.AcademicYearID,
		&item.BranchID,
		&item.Status,
		&item.IssueDate,
		&item.Currency,
		&item.AmountPaise,
		&item.PaymentMethod,
		&item.IssuedBy,
		&item.IssuedAt,
		&item.CancelledAt,
		&metadata,
		&item.CreatedAt,
		&item.UpdatedAt,
		&student.ID,
		&student.AdmissionNumber,
		&student.FirstName,
		&student.LastName,
		&student.Status,
		&academicYear.ID,
		&academicYear.Name,
		&academicYear.Code,
	); err != nil {
		return nil, err
	}
	item.Metadata = parseJSON(metadata)
	item.Student = &student
	item.AcademicYear = &academicYear
	return &item, nil
}

func scanOfflinePaymentReferenceScanner(row rowScanner) (*model.OfflinePaymentReference, error) {
	var item model.OfflinePaymentReference
	var metadata []byte
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.PaymentID,
		&item.PaymentMethod,
		&item.ReferenceNumber,
		&item.BankName,
		&item.InstrumentDate,
		&item.DepositedAt,
		&item.ClearanceStatus,
		&item.Remarks,
		&metadata,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.Metadata = parseJSON(metadata)
	return &item, nil
}

func scanPaymentEventScanner(row rowScanner) (*model.PaymentEvent, error) {
	var item model.PaymentEvent
	var metadata []byte
	student := model.Student{}
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.PaymentID,
		&item.AttemptID,
		&item.ReceiptID,
		&item.StudentID,
		&item.EventType,
		&item.Status,
		&item.AmountPaise,
		&item.Message,
		&metadata,
		&item.OccurredAt,
		&item.CreatedAt,
		&student.ID,
		&student.AdmissionNumber,
		&student.FirstName,
		&student.LastName,
		&student.Status,
	); err != nil {
		return nil, err
	}
	item.Metadata = parseJSON(metadata)
	item.Student = &student
	return &item, nil
}

func scanInvoiceTail(row rowScanner, prefixDest ...any) (*model.Invoice, error) {
	invoice := model.Invoice{}
	var metadata []byte
	student := model.Student{}
	academicYear := model.AcademicYear{}
	class := model.Class{}
	section := model.Section{}
	dest := append(prefixDest,
		&invoice.ID,
		&invoice.TenantID,
		&invoice.InvoiceNumber,
		&invoice.StudentID,
		&invoice.AcademicYearID,
		&invoice.ClassID,
		&invoice.SectionID,
		&invoice.FeeStructureID,
		&invoice.AssignmentID,
		&invoice.IssueDate,
		&invoice.DueDate,
		&invoice.BillingPeriodStart,
		&invoice.BillingPeriodEnd,
		&invoice.GenerationKey,
		&invoice.Status,
		&invoice.Currency,
		&invoice.AllowPartialPayment,
		&invoice.MinimumPartialAmountPaise,
		&invoice.SubtotalAmountPaise,
		&invoice.DiscountAmountPaise,
		&invoice.FineAmountPaise,
		&invoice.TaxAmountPaise,
		&invoice.TotalAmountPaise,
		&invoice.PaidAmountPaise,
		&invoice.BalanceAmountPaise,
		&invoice.GeneratedBy,
		&metadata,
		&invoice.IssuedAt,
		&invoice.CancelledAt,
		&invoice.VoidedAt,
		&invoice.CreatedAt,
		&invoice.UpdatedAt,
		&student.ID,
		&student.AdmissionNumber,
		&student.FirstName,
		&student.LastName,
		&student.Status,
		&academicYear.ID,
		&academicYear.Name,
		&academicYear.Code,
		&class.ID,
		&class.Name,
		&class.Code,
		&section.ID,
		&section.Name,
		&section.Code,
	)
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	invoice.Metadata = parseJSON(metadata)
	invoice.Student = &student
	invoice.AcademicYear = &academicYear
	invoice.Class = &class
	invoice.Section = &section
	return &invoice, nil
}

func paymentWhere(tenantID uuid.UUID, filter model.PaymentFilter) (string, []any) {
	clauses := []string{"p.tenant_id = $1"}
	args := []any{tenantID}
	if filter.StudentID != nil {
		args = append(args, *filter.StudentID)
		clauses = append(clauses, fmt.Sprintf("p.student_id = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Status) != "" {
		args = append(args, strings.TrimSpace(filter.Status))
		clauses = append(clauses, fmt.Sprintf("p.status = $%d", len(args)))
	}
	if strings.TrimSpace(filter.PaymentMethod) != "" {
		args = append(args, strings.TrimSpace(filter.PaymentMethod))
		clauses = append(clauses, fmt.Sprintf("p.payment_method = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Provider) != "" {
		args = append(args, strings.TrimSpace(filter.Provider))
		clauses = append(clauses, fmt.Sprintf("p.provider = $%d", len(args)))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		clauses = append(clauses, fmt.Sprintf("COALESCE(p.paid_at, p.created_at) >= $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		clauses = append(clauses, fmt.Sprintf("COALESCE(p.paid_at, p.created_at) <= $%d", len(args)))
	}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Search))+"%")
		clauses = append(clauses, fmt.Sprintf("(lower(p.id::text) LIKE $%d OR lower(coalesce(p.gateway_order_id, '')) LIKE $%d OR lower(coalesce(p.gateway_payment_id, '')) LIKE $%d OR lower(p.external_reference) LIKE $%d OR lower(p.settlement_reference) LIKE $%d OR lower(s.admission_number) LIKE $%d OR lower(s.first_name) LIKE $%d OR lower(s.last_name) LIKE $%d)", len(args), len(args), len(args), len(args), len(args), len(args), len(args), len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func receiptWhere(tenantID uuid.UUID, filter model.ReceiptFilter) (string, []any) {
	clauses := []string{"r.tenant_id = $1"}
	args := []any{tenantID}
	if filter.StudentID != nil {
		args = append(args, *filter.StudentID)
		clauses = append(clauses, fmt.Sprintf("r.student_id = $%d", len(args)))
	}
	if filter.GuardianUserID != nil {
		args = append(args, *filter.GuardianUserID)
		clauses = append(clauses, fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM student_guardians sg
			JOIN guardians g ON g.tenant_id = sg.tenant_id AND g.id = sg.guardian_id
			WHERE sg.tenant_id = r.tenant_id
				AND sg.student_id = r.student_id
				AND g.user_id = $%d
				AND g.deleted_at IS NULL
		)`, len(args)))
	}
	if strings.TrimSpace(filter.Status) != "" {
		args = append(args, strings.TrimSpace(filter.Status))
		clauses = append(clauses, fmt.Sprintf("r.status = $%d", len(args)))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		clauses = append(clauses, fmt.Sprintf("r.issue_date >= $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		clauses = append(clauses, fmt.Sprintf("r.issue_date <= $%d", len(args)))
	}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Search))+"%")
		clauses = append(clauses, fmt.Sprintf("(lower(r.receipt_number) LIKE $%d OR lower(s.admission_number) LIKE $%d OR lower(s.first_name) LIKE $%d OR lower(s.last_name) LIKE $%d)", len(args), len(args), len(args), len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func paymentEventWhere(tenantID uuid.UUID, filter model.PaymentEventFilter) (string, []any) {
	clauses := []string{"pe.tenant_id = $1"}
	args := []any{tenantID}
	if filter.StudentID != nil {
		args = append(args, *filter.StudentID)
		clauses = append(clauses, fmt.Sprintf("pe.student_id = $%d", len(args)))
	}
	if strings.TrimSpace(filter.EventType) != "" {
		args = append(args, strings.TrimSpace(filter.EventType))
		clauses = append(clauses, fmt.Sprintf("pe.event_type = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Status) != "" {
		args = append(args, strings.TrimSpace(filter.Status))
		clauses = append(clauses, fmt.Sprintf("pe.status = $%d", len(args)))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		clauses = append(clauses, fmt.Sprintf("pe.occurred_at >= $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		clauses = append(clauses, fmt.Sprintf("pe.occurred_at <= $%d", len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func receiptSeriesScopeKey(academicYearID uuid.UUID, branchID *uuid.UUID, prefix string) string {
	branch := "none"
	if branchID != nil {
		branch = branchID.String()
	}
	return academicYearID.String() + ":" + branch + ":" + strings.ToUpper(strings.TrimSpace(prefix))
}
