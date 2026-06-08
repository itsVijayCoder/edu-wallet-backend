package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apperror"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository"
)

const (
	paymentStatusSuccess          = "success"
	paymentStatusManuallyVerified = "manually_verified"
	paymentProviderOffline        = "offline"
)

type paymentService struct {
	repo         repository.PaymentRepository
	repoFactory  repository.PaymentRepositoryFactory
	academicRepo repository.AcademicRepository
	tx           database.Transactor
	auditRepo    repository.AuditRepository
	provider     PaymentProvider
	renderer     ReceiptRenderer
}

func NewPaymentService(
	repo repository.PaymentRepository,
	repoFactory repository.PaymentRepositoryFactory,
	academicRepo repository.AcademicRepository,
	tx database.Transactor,
	auditRepo repository.AuditRepository,
	provider PaymentProvider,
	renderer ReceiptRenderer,
) PaymentService {
	if provider == nil {
		provider = NewFakePaymentProvider("fake", "")
	}
	if renderer == nil {
		renderer = NewPDFReceiptRenderer()
	}
	return &paymentService{
		repo:         repo,
		repoFactory:  repoFactory,
		academicRepo: academicRepo,
		tx:           tx,
		auditRepo:    auditRepo,
		provider:     provider,
		renderer:     renderer,
	}
}

func (s *paymentService) CreatePaymentOrder(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreatePaymentOrderRequest) (*dto.PaymentOrderResponse, error) {
	if _, err := s.requireStudent(ctx, tenantID, req.StudentID); err != nil {
		return nil, err
	}
	invoiceIDs, err := normalizeInvoiceIDs(req.InvoiceIDs)
	if err != nil {
		return nil, err
	}
	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repo.GetPaymentAttemptByIdempotencyKey(ctx, tenantID, idempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("lookup payment idempotency key: %w", err)
		}
		if existing != nil && existing.ProviderOrderID != nil {
			resp := paymentOrderResponse(existing)
			return &resp, nil
		}
	}

	var attempt model.PaymentAttempt
	err = s.runInTx(ctx, func(repo repository.PaymentRepository) error {
		if idempotencyKey != "" {
			existing, err := repo.GetPaymentAttemptByIdempotencyKey(ctx, tenantID, idempotencyKey)
			if err != nil {
				return fmt.Errorf("lookup payment idempotency key: %w", err)
			}
			if existing != nil {
				attempt = *existing
				return nil
			}
		}

		allocations, total, err := s.buildInvoiceAllocations(ctx, repo, tenantID, req.StudentID, invoiceIDs, req.AmountPaise)
		if err != nil {
			return err
		}
		attempt = model.PaymentAttempt{
			TenantID:    tenantID,
			StudentID:   req.StudentID,
			Provider:    s.provider.Name(),
			Status:      "created",
			AmountPaise: total,
			Currency:    defaultCurrency,
			CreatedBy:   &actorID,
			Metadata:    normalizeMetadata(req.Metadata),
			Allocations: allocations,
		}
		if idempotencyKey != "" {
			attempt.IdempotencyKey = &idempotencyKey
		}
		if err := repo.CreatePaymentAttempt(ctx, &attempt); err != nil {
			return mapPersistenceError(err, "create payment attempt")
		}
		for i := range allocations {
			allocations[i].AttemptID = attempt.ID
		}
		if err := repo.CreatePaymentAttemptAllocations(ctx, allocations); err != nil {
			return mapPersistenceError(err, "create payment attempt allocations")
		}
		attempt.Allocations = allocations
		return nil
	})
	if err != nil {
		return nil, err
	}
	if attempt.ProviderOrderID == nil {
		order, err := s.provider.CreateOrder(ctx, PaymentOrderCreateRequest{
			AttemptID:   attempt.ID,
			TenantID:    tenantID,
			StudentID:   attempt.StudentID,
			AmountPaise: attempt.AmountPaise,
			Currency:    attempt.Currency,
			Receipt:     "attempt-" + attempt.ID.String(),
			Notes: map[string]string{
				"tenant_id":  tenantID.String(),
				"student_id": attempt.StudentID.String(),
				"attempt_id": attempt.ID.String(),
			},
		})
		if err != nil {
			_ = s.repo.UpdatePaymentAttemptStatus(ctx, tenantID, attempt.ID, "failed")
			return nil, err
		}
		if order.AmountPaise != attempt.AmountPaise || !strings.EqualFold(order.Currency, attempt.Currency) {
			_ = s.repo.UpdatePaymentAttemptStatus(ctx, tenantID, attempt.ID, "reconciliation_mismatch")
			return nil, apperror.New("PAYMENT_PROVIDER_AMOUNT_MISMATCH", "payment provider returned a mismatched order amount", http.StatusBadGateway)
		}
		if err := s.repo.UpdatePaymentAttemptProviderOrder(ctx, tenantID, attempt.ID, order.OrderID, order.CheckoutURL, "pending", order.Metadata); err != nil {
			return nil, fmt.Errorf("update payment attempt order: %w", err)
		}
		refreshed, err := s.repo.GetPaymentAttempt(ctx, tenantID, attempt.ID)
		if err != nil {
			return nil, fmt.Errorf("get payment attempt: %w", err)
		}
		attempt = *refreshed
	}
	if err := s.audit(ctx, tenantID, actorID, "payment_attempt.created", "payment_attempt", attempt.ID, "payment order created", map[string]any{
		"amount_paise": attempt.AmountPaise,
		"provider":     attempt.Provider,
	}); err != nil {
		return nil, err
	}
	resp := paymentOrderResponse(&attempt)
	return &resp, nil
}

func (s *paymentService) VerifyPayment(ctx context.Context, actorID, tenantID uuid.UUID, req dto.VerifyPaymentRequest) (*dto.PaymentVerificationResponse, error) {
	attempt, err := s.repo.GetPaymentAttemptByProviderOrderID(ctx, tenantID, s.provider.Name(), strings.TrimSpace(req.ProviderOrderID))
	if err != nil {
		return nil, fmt.Errorf("get payment attempt: %w", err)
	}
	if attempt == nil {
		return nil, apperror.ErrNotFound
	}
	if err := s.provider.VerifyPaymentSignature(PaymentSignatureVerification{
		OrderID:   strings.TrimSpace(req.ProviderOrderID),
		PaymentID: strings.TrimSpace(req.ProviderPaymentID),
		Signature: strings.TrimSpace(req.Signature),
	}); err != nil {
		return nil, err
	}

	paymentMethod := defaultString(req.PaymentMethod, "online")
	actor := actorID
	resp, err := s.finalizeSuccessfulAttempt(ctx, &actor, attempt, strings.TrimSpace(req.ProviderPaymentID), strings.TrimSpace(req.Signature), paymentMethod, paymentStatusSuccess, normalizeMetadata(req.Metadata))
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *paymentService) ProcessRazorpayWebhook(ctx context.Context, payload []byte, signature, eventID string) (*dto.WebhookProcessResponse, error) {
	if strings.TrimSpace(eventID) == "" {
		return nil, apperror.New("PAYMENT_WEBHOOK_EVENT_ID_REQUIRED", "webhook event id is required", http.StatusBadRequest)
	}
	if err := s.provider.VerifyWebhookSignature(payload, signature); err != nil {
		return nil, err
	}
	event, err := parseGatewayPaymentWebhook(payload)
	if err != nil {
		return nil, err
	}
	if event.ProviderOrderID == "" {
		return nil, apperror.New("PAYMENT_WEBHOOK_ORDER_REQUIRED", "webhook payload did not include a provider order id", http.StatusBadRequest)
	}
	if event.Outcome == "success" && event.ProviderPaymentID == "" {
		return nil, apperror.New("PAYMENT_WEBHOOK_PAYMENT_REQUIRED", "successful webhook payload did not include a provider payment id", http.StatusBadRequest)
	}
	attempt, err := s.repo.GetPaymentAttemptByProviderOrderIDAnyTenant(ctx, s.provider.Name(), event.ProviderOrderID)
	if err != nil {
		return nil, fmt.Errorf("get webhook payment attempt: %w", err)
	}
	if attempt == nil {
		return nil, apperror.ErrNotFound
	}
	if event.AmountPaise > 0 && event.AmountPaise != attempt.AmountPaise {
		return nil, apperror.New("PAYMENT_WEBHOOK_AMOUNT_MISMATCH", "webhook payment amount does not match the payment attempt", http.StatusBadRequest)
	}
	existingWebhook, err := s.repo.GetGatewayWebhookByEventID(ctx, s.provider.Name(), eventID)
	if err != nil {
		return nil, fmt.Errorf("lookup webhook event: %w", err)
	}
	if existingWebhook != nil {
		return &dto.WebhookProcessResponse{EventID: eventID, EventType: existingWebhook.EventType, Status: "duplicate"}, nil
	}

	webhook := &model.GatewayWebhook{
		TenantID:         attempt.TenantID,
		Provider:         s.provider.Name(),
		EventID:          eventID,
		EventType:        event.EventType,
		ProcessingStatus: "received",
		Payload:          event.Raw,
		Signature:        signature,
	}
	if err := s.repo.CreateGatewayWebhook(ctx, webhook); err != nil {
		return nil, mapPersistenceError(err, "create gateway webhook")
	}

	switch event.Outcome {
	case "success":
		resp, err := s.finalizeSuccessfulAttempt(ctx, nil, attempt, event.ProviderPaymentID, signature, defaultString(event.PaymentMethod, "online"), paymentStatusSuccess, map[string]any{"webhook_event_id": eventID, "event_type": event.EventType})
		if err != nil {
			_ = s.repo.UpdateGatewayWebhookStatus(ctx, attempt.TenantID, webhook.ID, "failed", err.Error())
			return nil, err
		}
		if err := s.repo.UpdateGatewayWebhookStatus(ctx, attempt.TenantID, webhook.ID, "processed", ""); err != nil {
			return nil, fmt.Errorf("mark webhook processed: %w", err)
		}
		return &dto.WebhookProcessResponse{
			EventID:   eventID,
			EventType: event.EventType,
			Status:    "processed",
			Payment:   &resp.Payment,
			Receipt:   resp.Receipt,
		}, nil
	case "failed":
		if err := s.repo.UpdatePaymentAttemptStatus(ctx, attempt.TenantID, attempt.ID, "failed"); err != nil {
			return nil, fmt.Errorf("mark attempt failed: %w", err)
		}
		if err := s.recordPaymentEvent(ctx, s.repo, attempt.TenantID, nil, &attempt.ID, nil, attempt.StudentID, "payment.failed", "failed", attempt.AmountPaise, "payment failed at provider", map[string]any{"webhook_event_id": eventID}); err != nil {
			return nil, err
		}
		if err := s.repo.UpdateGatewayWebhookStatus(ctx, attempt.TenantID, webhook.ID, "processed", ""); err != nil {
			return nil, fmt.Errorf("mark webhook processed: %w", err)
		}
		return &dto.WebhookProcessResponse{EventID: eventID, EventType: event.EventType, Status: "processed"}, nil
	default:
		if err := s.repo.UpdateGatewayWebhookStatus(ctx, attempt.TenantID, webhook.ID, "ignored", "event type ignored"); err != nil {
			return nil, fmt.Errorf("mark webhook ignored: %w", err)
		}
		return &dto.WebhookProcessResponse{EventID: eventID, EventType: event.EventType, Status: "ignored"}, nil
	}
}

func (s *paymentService) CreateOfflinePayment(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateOfflinePaymentRequest) (*dto.PaymentVerificationResponse, error) {
	if _, err := s.requireStudent(ctx, tenantID, req.StudentID); err != nil {
		return nil, err
	}
	receivedOn := time.Now().UTC()
	var err error
	if strings.TrimSpace(req.ReceivedOn) != "" {
		parsed, err := parseDate(req.ReceivedOn)
		if err != nil {
			return nil, err
		}
		receivedOn = parsed
	}
	instrumentDate, err := parseDatePointer(req.InstrumentDate)
	if err != nil {
		return nil, err
	}
	clearanceStatus := defaultString(req.ClearanceStatus, "cleared")
	if clearanceStatus != "cleared" {
		return nil, apperror.New("OFFLINE_PAYMENT_NOT_CLEARED", "only cleared offline payments can update invoice balances in this phase", http.StatusBadRequest)
	}

	var payment model.Payment
	var receipt *model.Receipt
	err = s.runInTx(ctx, func(repo repository.PaymentRepository) error {
		allocations, total, err := s.buildExplicitAllocations(ctx, repo, tenantID, req.StudentID, req.Allocations)
		if err != nil {
			return err
		}
		payment = model.Payment{
			TenantID:           tenantID,
			StudentID:          req.StudentID,
			Provider:           paymentProviderOffline,
			PaymentMethod:      strings.TrimSpace(req.PaymentMethod),
			Status:             paymentStatusManuallyVerified,
			AmountPaise:        total,
			AmountAppliedPaise: total,
			Currency:           defaultCurrency,
			ExternalReference:  strings.TrimSpace(req.ReferenceNumber),
			PaidAt:             &receivedOn,
			VerifiedAt:         &receivedOn,
			ReceivedBy:         &actorID,
			Metadata:           normalizeMetadata(req.Metadata),
		}
		if err := repo.CreatePayment(ctx, &payment); err != nil {
			return mapPersistenceError(err, "create offline payment")
		}
		for i := range allocations {
			if _, err := repo.ApplyInvoicePayment(ctx, tenantID, allocations[i].InvoiceID, allocations[i].AmountPaise, receivedOn); err != nil {
				return mapPersistenceError(err, "apply invoice payment")
			}
			allocations[i].PaymentID = payment.ID
		}
		if err := repo.CreatePaymentAllocations(ctx, allocations); err != nil {
			return mapPersistenceError(err, "create payment allocations")
		}
		payment.Allocations = allocations
		ref := &model.OfflinePaymentReference{
			TenantID:        tenantID,
			PaymentID:       payment.ID,
			PaymentMethod:   payment.PaymentMethod,
			ReferenceNumber: strings.TrimSpace(req.ReferenceNumber),
			BankName:        strings.TrimSpace(req.BankName),
			InstrumentDate:  instrumentDate,
			ClearanceStatus: clearanceStatus,
			Remarks:         strings.TrimSpace(req.Remarks),
			Metadata:        normalizeMetadata(req.Metadata),
		}
		if err := repo.CreateOfflinePaymentReference(ctx, ref); err != nil {
			return mapPersistenceError(err, "create offline payment reference")
		}
		receipt, err = s.ensureReceipt(ctx, repo, &payment, allocations, &actorID, receivedOn)
		if err != nil {
			return err
		}
		payment.Receipt = receipt
		if err := s.recordPaymentEvent(ctx, repo, tenantID, &payment.ID, nil, &receipt.ID, req.StudentID, "payment.offline_recorded", payment.Status, total, "offline payment recorded", map[string]any{"payment_method": payment.PaymentMethod}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.audit(ctx, tenantID, actorID, "payment.offline_recorded", "payment", payment.ID, "offline payment recorded", map[string]any{"amount_paise": payment.AmountPaise, "payment_method": payment.PaymentMethod}); err != nil {
		return nil, err
	}
	if receipt != nil {
		if err := s.audit(ctx, tenantID, actorID, "receipt.issued", "receipt", receipt.ID, "receipt issued", map[string]any{"receipt_number": receipt.ReceiptNumber}); err != nil {
			return nil, err
		}
	}
	paymentResp := paymentToResponse(&payment)
	receiptResp := receiptToResponse(receipt)
	return &dto.PaymentVerificationResponse{Payment: paymentResp, Receipt: receiptResp}, nil
}

func (s *paymentService) ListPayments(ctx context.Context, tenantID uuid.UUID, filter model.PaymentFilter, params model.PaginationParams) (*model.PaginatedResult[dto.PaymentResponse], error) {
	result, err := s.repo.ListPayments(ctx, tenantID, filter, params)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	items := make([]dto.PaymentResponse, len(result.Data))
	for i := range result.Data {
		items[i] = paymentToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *paymentService) GetPayment(ctx context.Context, tenantID, id uuid.UUID) (*dto.PaymentResponse, error) {
	payment, err := s.repo.GetPayment(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get payment: %w", err)
	}
	if payment == nil {
		return nil, apperror.ErrNotFound
	}
	resp := paymentToResponse(payment)
	return &resp, nil
}

func (s *paymentService) ListReceipts(ctx context.Context, tenantID uuid.UUID, filter model.ReceiptFilter, params model.PaginationParams) (*model.PaginatedResult[dto.ReceiptResponse], error) {
	result, err := s.repo.ListReceipts(ctx, tenantID, filter, params)
	if err != nil {
		return nil, fmt.Errorf("list receipts: %w", err)
	}
	items := make([]dto.ReceiptResponse, len(result.Data))
	for i := range result.Data {
		items[i] = *receiptToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *paymentService) GetReceipt(ctx context.Context, tenantID, id uuid.UUID) (*dto.ReceiptResponse, error) {
	receipt, err := s.repo.GetReceipt(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get receipt: %w", err)
	}
	if receipt == nil {
		return nil, apperror.ErrNotFound
	}
	return receiptToResponse(receipt), nil
}

func (s *paymentService) DownloadReceipt(ctx context.Context, tenantID, id uuid.UUID) (*dto.ReceiptDownloadResponse, error) {
	receipt, err := s.GetReceipt(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	return s.renderer.RenderReceipt(ctx, *receipt)
}

func (s *paymentService) ListParentReceipts(ctx context.Context, tenantID uuid.UUID, filter model.ReceiptFilter, params model.PaginationParams) (*model.PaginatedResult[dto.ReceiptResponse], error) {
	return s.ListReceipts(ctx, tenantID, filter, params)
}

func (s *paymentService) DownloadParentReceipt(ctx context.Context, tenantID, id uuid.UUID) (*dto.ReceiptDownloadResponse, error) {
	return s.DownloadReceipt(ctx, tenantID, id)
}

func (s *paymentService) ListPaymentEvents(ctx context.Context, tenantID uuid.UUID, filter model.PaymentEventFilter, params model.PaginationParams) (*model.PaginatedResult[dto.PaymentEventResponse], error) {
	result, err := s.repo.ListPaymentEvents(ctx, tenantID, filter, params)
	if err != nil {
		return nil, fmt.Errorf("list payment events: %w", err)
	}
	items := make([]dto.PaymentEventResponse, len(result.Data))
	for i := range result.Data {
		items[i] = paymentEventToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *paymentService) finalizeSuccessfulAttempt(ctx context.Context, actorID *uuid.UUID, attempt *model.PaymentAttempt, providerPaymentID, signature, paymentMethod, status string, metadata map[string]any) (*dto.PaymentVerificationResponse, error) {
	existing, err := s.repo.GetPaymentByGatewayPaymentID(ctx, attempt.TenantID, attempt.Provider, providerPaymentID)
	if err != nil {
		return nil, fmt.Errorf("lookup provider payment: %w", err)
	}
	if existing != nil {
		paymentResp := paymentToResponse(existing)
		receiptResp := receiptToResponse(existing.Receipt)
		return &dto.PaymentVerificationResponse{Payment: paymentResp, Receipt: receiptResp}, nil
	}

	var payment model.Payment
	var receipt *model.Receipt
	now := time.Now().UTC()
	err = s.runInTx(ctx, func(repo repository.PaymentRepository) error {
		txAttempt, err := repo.GetPaymentAttempt(ctx, attempt.TenantID, attempt.ID)
		if err != nil {
			return fmt.Errorf("get payment attempt: %w", err)
		}
		if txAttempt == nil {
			return apperror.ErrNotFound
		}
		existing, err := repo.GetPaymentByGatewayPaymentID(ctx, attempt.TenantID, attempt.Provider, providerPaymentID)
		if err != nil {
			return fmt.Errorf("lookup provider payment: %w", err)
		}
		if existing != nil {
			payment = *existing
			receipt = existing.Receipt
			return nil
		}
		if len(txAttempt.Allocations) == 0 {
			return apperror.New("PAYMENT_ATTEMPT_ALLOCATIONS_REQUIRED", "payment attempt has no invoice allocations", http.StatusConflict)
		}
		allocations := make([]model.PaymentAllocation, len(txAttempt.Allocations))
		total := int64(0)
		for i, allocation := range txAttempt.Allocations {
			invoice, err := repo.GetInvoiceForPayment(ctx, txAttempt.TenantID, allocation.InvoiceID)
			if err != nil {
				return fmt.Errorf("lock invoice: %w", err)
			}
			if invoice == nil {
				return apperror.ErrNotFound
			}
			if invoice.BalanceAmountPaise < allocation.AmountPaise {
				return apperror.New("INVOICE_BALANCE_CHANGED", "invoice balance is lower than the payment allocation", http.StatusConflict)
			}
			allocations[i] = allocation
			allocations[i].Invoice = invoice
			total += allocation.AmountPaise
		}
		if total != txAttempt.AmountPaise {
			return apperror.New("PAYMENT_ATTEMPT_AMOUNT_MISMATCH", "payment attempt amount does not match invoice allocations", http.StatusConflict)
		}
		orderID := ""
		if txAttempt.ProviderOrderID != nil {
			orderID = *txAttempt.ProviderOrderID
		}
		paymentID := providerPaymentID
		payment = model.Payment{
			TenantID:           txAttempt.TenantID,
			AttemptID:          &txAttempt.ID,
			StudentID:          txAttempt.StudentID,
			Provider:           txAttempt.Provider,
			PaymentMethod:      defaultString(paymentMethod, "online"),
			Status:             status,
			AmountPaise:        txAttempt.AmountPaise,
			AmountAppliedPaise: txAttempt.AmountPaise,
			Currency:           txAttempt.Currency,
			GatewayOrderID:     &orderID,
			GatewayPaymentID:   &paymentID,
			GatewaySignature:   signature,
			PaidAt:             &now,
			VerifiedAt:         &now,
			ReceivedBy:         actorID,
			Metadata:           normalizeMetadata(metadata),
		}
		if err := repo.CreatePayment(ctx, &payment); err != nil {
			return mapPersistenceError(err, "create payment")
		}
		for i := range allocations {
			if _, err := repo.ApplyInvoicePayment(ctx, txAttempt.TenantID, allocations[i].InvoiceID, allocations[i].AmountPaise, now); err != nil {
				return mapPersistenceError(err, "apply invoice payment")
			}
			allocations[i].PaymentID = payment.ID
		}
		if err := repo.CreatePaymentAllocations(ctx, allocations); err != nil {
			return mapPersistenceError(err, "create payment allocations")
		}
		payment.Allocations = allocations
		if err := repo.UpdatePaymentAttemptStatus(ctx, txAttempt.TenantID, txAttempt.ID, paymentStatusSuccess); err != nil {
			return fmt.Errorf("mark payment attempt successful: %w", err)
		}
		receipt, err = s.ensureReceipt(ctx, repo, &payment, allocations, actorID, now)
		if err != nil {
			return err
		}
		payment.Receipt = receipt
		if err := s.recordPaymentEvent(ctx, repo, txAttempt.TenantID, &payment.ID, &txAttempt.ID, &receipt.ID, txAttempt.StudentID, "payment.succeeded", payment.Status, payment.AmountPaise, "payment succeeded", map[string]any{"provider": txAttempt.Provider}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	auditActor := uuid.Nil
	if actorID != nil {
		auditActor = *actorID
	}
	if err := s.audit(ctx, payment.TenantID, auditActor, "payment.received", "payment", payment.ID, "payment received", map[string]any{"amount_paise": payment.AmountPaise, "provider": payment.Provider}); err != nil {
		return nil, err
	}
	if receipt != nil {
		if err := s.audit(ctx, payment.TenantID, auditActor, "receipt.issued", "receipt", receipt.ID, "receipt issued", map[string]any{"receipt_number": receipt.ReceiptNumber}); err != nil {
			return nil, err
		}
	}
	paymentResp := paymentToResponse(&payment)
	receiptResp := receiptToResponse(receipt)
	return &dto.PaymentVerificationResponse{Payment: paymentResp, Receipt: receiptResp}, nil
}

func (s *paymentService) buildInvoiceAllocations(ctx context.Context, repo repository.PaymentRepository, tenantID, studentID uuid.UUID, invoiceIDs []uuid.UUID, requestedAmount int64) ([]model.PaymentAllocation, int64, error) {
	invoices := make([]model.Invoice, 0, len(invoiceIDs))
	totalBalance := int64(0)
	for _, invoiceID := range invoiceIDs {
		invoice, err := repo.GetInvoiceForPayment(ctx, tenantID, invoiceID)
		if err != nil {
			return nil, 0, fmt.Errorf("lock invoice: %w", err)
		}
		if invoice == nil {
			return nil, 0, apperror.ErrNotFound
		}
		if invoice.StudentID != studentID {
			return nil, 0, apperror.New("INVOICE_STUDENT_MISMATCH", "invoice does not belong to the selected student", http.StatusBadRequest)
		}
		if invoice.Status == "cancelled" || invoice.Status == "void" {
			return nil, 0, apperror.New("INVOICE_NOT_PAYABLE", "cancelled or void invoices cannot be paid", http.StatusConflict)
		}
		if invoice.BalanceAmountPaise <= 0 {
			return nil, 0, apperror.New("INVOICE_ALREADY_PAID", "invoice has no payable balance", http.StatusConflict)
		}
		invoices = append(invoices, *invoice)
		totalBalance += invoice.BalanceAmountPaise
	}
	payAmount := requestedAmount
	if payAmount == 0 {
		payAmount = totalBalance
	}
	if payAmount <= 0 || payAmount > totalBalance {
		return nil, 0, apperror.New("INVALID_PAYMENT_AMOUNT", "payment amount must be greater than zero and cannot exceed selected invoice balance", http.StatusBadRequest)
	}
	if payAmount < totalBalance && len(invoices) > 1 {
		return nil, 0, apperror.New("PARTIAL_MULTI_INVOICE_UNSUPPORTED", "partial payment is supported for one invoice at a time", http.StatusBadRequest)
	}

	allocations := make([]model.PaymentAllocation, 0, len(invoices))
	remaining := payAmount
	for i := range invoices {
		invoice := invoices[i]
		amount := invoice.BalanceAmountPaise
		if remaining < amount {
			amount = remaining
		}
		if amount < invoice.BalanceAmountPaise {
			if err := validateInvoicePartialPayment(invoice, amount); err != nil {
				return nil, 0, err
			}
		}
		allocations = append(allocations, model.PaymentAllocation{
			TenantID:    tenantID,
			InvoiceID:   invoice.ID,
			AmountPaise: amount,
			Invoice:     &invoice,
		})
		remaining -= amount
		if remaining == 0 {
			break
		}
	}
	return allocations, payAmount, nil
}

func (s *paymentService) buildExplicitAllocations(ctx context.Context, repo repository.PaymentRepository, tenantID, studentID uuid.UUID, reqs []dto.OfflinePaymentAllocationRequest) ([]model.PaymentAllocation, int64, error) {
	if len(reqs) == 0 {
		return nil, 0, apperror.New("PAYMENT_ALLOCATIONS_REQUIRED", "at least one invoice allocation is required", http.StatusBadRequest)
	}
	seen := map[uuid.UUID]bool{}
	allocations := make([]model.PaymentAllocation, 0, len(reqs))
	total := int64(0)
	for _, req := range reqs {
		if req.InvoiceID == uuid.Nil {
			return nil, 0, apperror.New("INVALID_INVOICE_ID", "invoice_id cannot be empty", http.StatusBadRequest)
		}
		if req.AmountPaise <= 0 {
			return nil, 0, apperror.New("INVALID_PAYMENT_AMOUNT", "allocation amount must be greater than zero", http.StatusBadRequest)
		}
		if seen[req.InvoiceID] {
			return nil, 0, apperror.New("DUPLICATE_INVOICE_ALLOCATION", "invoice allocations cannot contain duplicates", http.StatusBadRequest)
		}
		seen[req.InvoiceID] = true
		invoice, err := repo.GetInvoiceForPayment(ctx, tenantID, req.InvoiceID)
		if err != nil {
			return nil, 0, fmt.Errorf("lock invoice: %w", err)
		}
		if invoice == nil {
			return nil, 0, apperror.ErrNotFound
		}
		if invoice.StudentID != studentID {
			return nil, 0, apperror.New("INVOICE_STUDENT_MISMATCH", "invoice does not belong to the selected student", http.StatusBadRequest)
		}
		if req.AmountPaise > invoice.BalanceAmountPaise {
			return nil, 0, apperror.New("INVALID_PAYMENT_AMOUNT", "allocation amount cannot exceed invoice balance", http.StatusBadRequest)
		}
		if req.AmountPaise < invoice.BalanceAmountPaise {
			if err := validateInvoicePartialPayment(*invoice, req.AmountPaise); err != nil {
				return nil, 0, err
			}
		}
		allocations = append(allocations, model.PaymentAllocation{
			TenantID:    tenantID,
			InvoiceID:   invoice.ID,
			AmountPaise: req.AmountPaise,
			Invoice:     invoice,
		})
		total += req.AmountPaise
	}
	return allocations, total, nil
}

func (s *paymentService) ensureReceipt(ctx context.Context, repo repository.PaymentRepository, payment *model.Payment, allocations []model.PaymentAllocation, actorID *uuid.UUID, issuedAt time.Time) (*model.Receipt, error) {
	existing, err := repo.GetReceiptByPaymentID(ctx, payment.TenantID, payment.ID)
	if err != nil {
		return nil, fmt.Errorf("lookup receipt by payment: %w", err)
	}
	if existing != nil {
		return existing, nil
	}
	if len(allocations) == 0 || allocations[0].Invoice == nil {
		return nil, apperror.New("RECEIPT_ALLOCATIONS_REQUIRED", "receipt requires at least one invoice allocation", http.StatusConflict)
	}
	firstInvoice := allocations[0].Invoice
	prefix := receiptPrefix(firstInvoice)
	sequence, err := repo.NextReceiptNumber(ctx, payment.TenantID, firstInvoice.AcademicYearID, nil, prefix)
	if err != nil {
		return nil, fmt.Errorf("next receipt number: %w", err)
	}
	receipt := &model.Receipt{
		TenantID:       payment.TenantID,
		ReceiptNumber:  fmt.Sprintf("%s-%06d", prefix, sequence),
		PaymentID:      payment.ID,
		StudentID:      payment.StudentID,
		AcademicYearID: firstInvoice.AcademicYearID,
		Status:         "issued",
		IssueDate:      issuedAt,
		Currency:       payment.Currency,
		AmountPaise:    payment.AmountAppliedPaise,
		PaymentMethod:  payment.PaymentMethod,
		IssuedBy:       actorID,
		IssuedAt:       issuedAt,
		Metadata:       map[string]any{"payment_id": payment.ID.String()},
		Allocations:    allocations,
	}
	if err := repo.CreateReceipt(ctx, receipt); err != nil {
		return nil, mapPersistenceError(err, "create receipt")
	}
	return receipt, nil
}

func (s *paymentService) requireStudent(ctx context.Context, tenantID, studentID uuid.UUID) (*model.Student, error) {
	student, err := s.academicRepo.GetStudent(ctx, tenantID, studentID)
	if err != nil {
		return nil, fmt.Errorf("get student: %w", err)
	}
	if student == nil {
		return nil, apperror.ErrNotFound
	}
	return student, nil
}

func (s *paymentService) recordPaymentEvent(ctx context.Context, repo repository.PaymentRepository, tenantID uuid.UUID, paymentID, attemptID, receiptID *uuid.UUID, studentID uuid.UUID, eventType, status string, amountPaise int64, message string, metadata map[string]any) error {
	event := &model.PaymentEvent{
		TenantID:    tenantID,
		PaymentID:   paymentID,
		AttemptID:   attemptID,
		ReceiptID:   receiptID,
		StudentID:   studentID,
		EventType:   eventType,
		Status:      status,
		AmountPaise: amountPaise,
		Message:     message,
		Metadata:    normalizeMetadata(metadata),
		OccurredAt:  time.Now().UTC(),
	}
	if err := repo.CreatePaymentEvent(ctx, event); err != nil {
		return fmt.Errorf("record payment event: %w", err)
	}
	return nil
}

func (s *paymentService) runInTx(ctx context.Context, fn func(repository.PaymentRepository) error) error {
	if s.tx == nil || s.repoFactory == nil {
		return fn(s.repo)
	}
	return s.tx.WithinTx(ctx, func(db database.DBTX) error {
		return fn(s.repoFactory(db))
	})
}

func (s *paymentService) audit(ctx context.Context, tenantID uuid.UUID, actorID uuid.UUID, action string, entityType string, entityID uuid.UUID, summary string, metadata map[string]any) error {
	if s.auditRepo == nil {
		return nil
	}
	var actor *uuid.UUID
	if actorID != uuid.Nil {
		actor = &actorID
	}
	entry := &model.AuditLog{
		TenantID:    &tenantID,
		ActorUserID: actor,
		Action:      action,
		EntityType:  entityType,
		EntityID:    &entityID,
		Summary:     summary,
		Metadata:    normalizeMetadata(metadata),
	}
	if err := s.auditRepo.Create(ctx, entry); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}

type gatewayPaymentWebhook struct {
	EventType         string
	ProviderOrderID   string
	ProviderPaymentID string
	PaymentMethod     string
	AmountPaise       int64
	Outcome           string
	Raw               map[string]any
}

func parseGatewayPaymentWebhook(payload []byte) (*gatewayPaymentWebhook, error) {
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, apperror.New("PAYMENT_WEBHOOK_INVALID_JSON", "webhook payload must be valid JSON", http.StatusBadRequest)
	}
	eventType, _ := raw["event"].(string)
	entity := nestedMap(raw, "payload", "payment", "entity")
	orderID, _ := entity["order_id"].(string)
	paymentID, _ := entity["id"].(string)
	method, _ := entity["method"].(string)
	status, _ := entity["status"].(string)
	captured, _ := entity["captured"].(bool)
	amount := numberToInt64(entity["amount"])

	outcome := "ignored"
	switch {
	case strings.Contains(eventType, "failed") || status == "failed":
		outcome = "failed"
	case strings.Contains(eventType, "captured") || strings.Contains(eventType, "authorized") || status == "captured" || captured:
		outcome = "success"
	}
	return &gatewayPaymentWebhook{
		EventType:         eventType,
		ProviderOrderID:   strings.TrimSpace(orderID),
		ProviderPaymentID: strings.TrimSpace(paymentID),
		PaymentMethod:     strings.TrimSpace(method),
		AmountPaise:       amount,
		Outcome:           outcome,
		Raw:               raw,
	}, nil
}

func nestedMap(root map[string]any, path ...string) map[string]any {
	current := root
	for _, key := range path {
		next, ok := current[key].(map[string]any)
		if !ok {
			return map[string]any{}
		}
		current = next
	}
	return current
}

func numberToInt64(value any) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	default:
		return 0
	}
}

func normalizeInvoiceIDs(ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, apperror.New("INVOICE_IDS_REQUIRED", "at least one invoice_id is required", http.StatusBadRequest)
	}
	seen := map[uuid.UUID]bool{}
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			return nil, apperror.New("INVALID_INVOICE_ID", "invoice_ids cannot contain an empty UUID", http.StatusBadRequest)
		}
		if seen[id] {
			return nil, apperror.New("DUPLICATE_INVOICE_ID", "invoice_ids cannot contain duplicates", http.StatusBadRequest)
		}
		seen[id] = true
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out, nil
}

func validateInvoicePartialPayment(invoice model.Invoice, amountPaise int64) error {
	if !invoice.AllowPartialPayment {
		return apperror.New("PARTIAL_PAYMENT_NOT_ALLOWED", "invoice does not allow partial payments", http.StatusBadRequest)
	}
	minimum := invoice.MinimumPartialAmountPaise
	if minimum > 0 && amountPaise < minimum {
		return apperror.New("PARTIAL_PAYMENT_BELOW_MINIMUM", "payment amount is below the invoice minimum partial amount", http.StatusBadRequest)
	}
	return nil
}

func receiptPrefix(invoice *model.Invoice) string {
	code := "GEN"
	if invoice.AcademicYear != nil && strings.TrimSpace(invoice.AcademicYear.Code) != "" {
		code = invoice.AcademicYear.Code
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	replacer := strings.NewReplacer(" ", "", "/", "-", "_", "-")
	return "REC-" + replacer.Replace(code)
}

func paymentOrderResponse(attempt *model.PaymentAttempt) dto.PaymentOrderResponse {
	attemptResp := paymentAttemptToResponse(attempt)
	orderID := ""
	if attempt.ProviderOrderID != nil {
		orderID = *attempt.ProviderOrderID
	}
	return dto.PaymentOrderResponse{
		Attempt:     attemptResp,
		OrderID:     orderID,
		AmountPaise: attempt.AmountPaise,
		Currency:    attempt.Currency,
		Provider:    attempt.Provider,
		CheckoutURL: attempt.CheckoutURL,
	}
}

func paymentAttemptToResponse(attempt *model.PaymentAttempt) dto.PaymentAttemptResponse {
	resp := dto.PaymentAttemptResponse{
		ID:                   attempt.ID,
		TenantID:             attempt.TenantID,
		StudentID:            attempt.StudentID,
		Provider:             attempt.Provider,
		ProviderOrderID:      attempt.ProviderOrderID,
		IdempotencyKey:       attempt.IdempotencyKey,
		Status:               attempt.Status,
		AmountPaise:          attempt.AmountPaise,
		Currency:             attempt.Currency,
		CheckoutURL:          attempt.CheckoutURL,
		ExpiresAt:            attempt.ExpiresAt,
		ProviderRetryCount:   attempt.ProviderRetryCount,
		ProviderLastError:    attempt.ProviderLastError,
		ReconciliationStatus: attempt.ReconciliationStatus,
		ReconciledAt:         attempt.ReconciledAt,
		SettlementReference:  attempt.SettlementReference,
		SettledAt:            attempt.SettledAt,
		Metadata:             attempt.Metadata,
		CreatedAt:            attempt.CreatedAt,
		UpdatedAt:            attempt.UpdatedAt,
	}
	if attempt.Student != nil {
		student := studentBriefToResponse(attempt.Student)
		resp.Student = &student
	}
	resp.Allocations = paymentAllocationsToResponse(attempt.Allocations)
	return resp
}

func paymentToResponse(payment *model.Payment) dto.PaymentResponse {
	if payment == nil {
		return dto.PaymentResponse{}
	}
	resp := dto.PaymentResponse{
		ID:                   payment.ID,
		TenantID:             payment.TenantID,
		AttemptID:            payment.AttemptID,
		StudentID:            payment.StudentID,
		Provider:             payment.Provider,
		PaymentMethod:        payment.PaymentMethod,
		Status:               payment.Status,
		AmountPaise:          payment.AmountPaise,
		AmountAppliedPaise:   payment.AmountAppliedPaise,
		Currency:             payment.Currency,
		GatewayOrderID:       payment.GatewayOrderID,
		GatewayPaymentID:     payment.GatewayPaymentID,
		ExternalReference:    payment.ExternalReference,
		PaidAt:               payment.PaidAt,
		VerifiedAt:           payment.VerifiedAt,
		ReconciliationStatus: payment.ReconciliationStatus,
		ReconciledAt:         payment.ReconciledAt,
		SettlementReference:  payment.SettlementReference,
		SettledAt:            payment.SettledAt,
		Metadata:             payment.Metadata,
		Allocations:          paymentAllocationsToResponse(payment.Allocations),
		CreatedAt:            payment.CreatedAt,
		UpdatedAt:            payment.UpdatedAt,
	}
	if payment.Student != nil {
		student := studentBriefToResponse(payment.Student)
		resp.Student = &student
	}
	if payment.Receipt != nil {
		resp.Receipt = receiptToResponse(payment.Receipt)
	}
	return resp
}

func receiptToResponse(receipt *model.Receipt) *dto.ReceiptResponse {
	if receipt == nil {
		return nil
	}
	resp := &dto.ReceiptResponse{
		ID:             receipt.ID,
		TenantID:       receipt.TenantID,
		ReceiptNumber:  receipt.ReceiptNumber,
		PaymentID:      receipt.PaymentID,
		StudentID:      receipt.StudentID,
		AcademicYearID: receipt.AcademicYearID,
		Status:         receipt.Status,
		IssueDate:      receipt.IssueDate.Format(dateLayout),
		Currency:       receipt.Currency,
		AmountPaise:    receipt.AmountPaise,
		PaymentMethod:  receipt.PaymentMethod,
		Metadata:       receipt.Metadata,
		Allocations:    paymentAllocationsToResponse(receipt.Allocations),
		IssuedAt:       receipt.IssuedAt,
		CreatedAt:      receipt.CreatedAt,
		UpdatedAt:      receipt.UpdatedAt,
	}
	if receipt.Student != nil {
		student := studentBriefToResponse(receipt.Student)
		resp.Student = &student
	}
	if receipt.AcademicYear != nil {
		resp.AcademicYear = &dto.LookupResponse{ID: receipt.AcademicYear.ID, Name: receipt.AcademicYear.Name, Code: receipt.AcademicYear.Code}
	}
	if receipt.Payment != nil {
		payment := *receipt.Payment
		payment.Receipt = nil
		payment.Allocations = nil
		paymentResp := paymentToResponse(&payment)
		resp.Payment = &paymentResp
	}
	return resp
}

func paymentAllocationsToResponse(allocations []model.PaymentAllocation) []dto.PaymentAllocationResponse {
	if len(allocations) == 0 {
		return nil
	}
	resp := make([]dto.PaymentAllocationResponse, len(allocations))
	for i := range allocations {
		resp[i] = dto.PaymentAllocationResponse{
			InvoiceID:   allocations[i].InvoiceID,
			AmountPaise: allocations[i].AmountPaise,
			CreatedAt:   allocations[i].CreatedAt,
		}
		if allocations[i].Invoice != nil {
			resp[i].InvoiceNumber = allocations[i].Invoice.InvoiceNumber
			invoice := invoiceToResponse(allocations[i].Invoice)
			resp[i].Invoice = &invoice
		}
	}
	return resp
}

func paymentEventToResponse(event *model.PaymentEvent) dto.PaymentEventResponse {
	resp := dto.PaymentEventResponse{
		ID:          event.ID,
		TenantID:    event.TenantID,
		PaymentID:   event.PaymentID,
		AttemptID:   event.AttemptID,
		ReceiptID:   event.ReceiptID,
		StudentID:   event.StudentID,
		EventType:   event.EventType,
		Status:      event.Status,
		AmountPaise: event.AmountPaise,
		Message:     event.Message,
		Metadata:    event.Metadata,
		OccurredAt:  event.OccurredAt,
		CreatedAt:   event.CreatedAt,
	}
	if event.Student != nil {
		student := studentBriefToResponse(event.Student)
		resp.Student = &student
	}
	return resp
}
