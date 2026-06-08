package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreatePaymentOrderRequest struct {
	StudentID      uuid.UUID      `json:"student_id" binding:"required"`
	InvoiceIDs     []uuid.UUID    `json:"invoice_ids" binding:"required,min=1,dive,required"`
	AmountPaise    int64          `json:"amount_paise" binding:"omitempty,min=1"`
	IdempotencyKey string         `json:"idempotency_key" binding:"omitempty,max=160"`
	Metadata       map[string]any `json:"metadata"`
}

type VerifyPaymentRequest struct {
	ProviderOrderID   string         `json:"provider_order_id" binding:"required"`
	ProviderPaymentID string         `json:"provider_payment_id" binding:"required"`
	Signature         string         `json:"signature" binding:"required"`
	PaymentMethod     string         `json:"payment_method" binding:"omitempty,oneof=online upi card netbanking wallet other"`
	Metadata          map[string]any `json:"metadata"`
}

type OfflinePaymentAllocationRequest struct {
	InvoiceID   uuid.UUID `json:"invoice_id" binding:"required"`
	AmountPaise int64     `json:"amount_paise" binding:"required,min=1"`
}

type CreateOfflinePaymentRequest struct {
	StudentID       uuid.UUID                         `json:"student_id" binding:"required"`
	PaymentMethod   string                            `json:"payment_method" binding:"required,oneof=cash cheque dd bank_transfer upi other"`
	Allocations     []OfflinePaymentAllocationRequest `json:"allocations" binding:"required,min=1,dive"`
	ReceivedOn      string                            `json:"received_on"`
	ReferenceNumber string                            `json:"reference_number" binding:"omitempty,max=160"`
	BankName        string                            `json:"bank_name" binding:"omitempty,max=160"`
	InstrumentDate  *string                           `json:"instrument_date"`
	ClearanceStatus string                            `json:"clearance_status" binding:"omitempty,oneof=pending cleared bounced cancelled"`
	Remarks         string                            `json:"remarks"`
	Metadata        map[string]any                    `json:"metadata"`
}

type PaymentAttemptResponse struct {
	ID              uuid.UUID                   `json:"id"`
	TenantID        uuid.UUID                   `json:"tenant_id"`
	StudentID       uuid.UUID                   `json:"student_id"`
	Provider        string                      `json:"provider"`
	ProviderOrderID *string                     `json:"provider_order_id,omitempty"`
	IdempotencyKey  *string                     `json:"idempotency_key,omitempty"`
	Status          string                      `json:"status"`
	AmountPaise     int64                       `json:"amount_paise"`
	Currency        string                      `json:"currency"`
	CheckoutURL     string                      `json:"checkout_url"`
	ExpiresAt       *time.Time                  `json:"expires_at,omitempty"`
	Metadata        map[string]any              `json:"metadata,omitempty"`
	Student         *StudentBriefResponse       `json:"student,omitempty"`
	Allocations     []PaymentAllocationResponse `json:"allocations,omitempty"`
	CreatedAt       time.Time                   `json:"created_at"`
	UpdatedAt       time.Time                   `json:"updated_at"`
}

type PaymentResponse struct {
	ID                 uuid.UUID                   `json:"id"`
	TenantID           uuid.UUID                   `json:"tenant_id"`
	AttemptID          *uuid.UUID                  `json:"attempt_id,omitempty"`
	StudentID          uuid.UUID                   `json:"student_id"`
	Provider           string                      `json:"provider"`
	PaymentMethod      string                      `json:"payment_method"`
	Status             string                      `json:"status"`
	AmountPaise        int64                       `json:"amount_paise"`
	AmountAppliedPaise int64                       `json:"amount_applied_paise"`
	Currency           string                      `json:"currency"`
	GatewayOrderID     *string                     `json:"gateway_order_id,omitempty"`
	GatewayPaymentID   *string                     `json:"gateway_payment_id,omitempty"`
	ExternalReference  string                      `json:"external_reference,omitempty"`
	PaidAt             *time.Time                  `json:"paid_at,omitempty"`
	VerifiedAt         *time.Time                  `json:"verified_at,omitempty"`
	Metadata           map[string]any              `json:"metadata,omitempty"`
	Student            *StudentBriefResponse       `json:"student,omitempty"`
	Allocations        []PaymentAllocationResponse `json:"allocations,omitempty"`
	Receipt            *ReceiptResponse            `json:"receipt,omitempty"`
	CreatedAt          time.Time                   `json:"created_at"`
	UpdatedAt          time.Time                   `json:"updated_at"`
}

type PaymentAllocationResponse struct {
	InvoiceID     uuid.UUID        `json:"invoice_id"`
	InvoiceNumber string           `json:"invoice_number,omitempty"`
	AmountPaise   int64            `json:"amount_paise"`
	Invoice       *InvoiceResponse `json:"invoice,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
}

type PaymentOrderResponse struct {
	Attempt     PaymentAttemptResponse `json:"attempt"`
	OrderID     string                 `json:"order_id"`
	AmountPaise int64                  `json:"amount_paise"`
	Currency    string                 `json:"currency"`
	Provider    string                 `json:"provider"`
	CheckoutURL string                 `json:"checkout_url"`
}

type PaymentVerificationResponse struct {
	Payment PaymentResponse  `json:"payment"`
	Receipt *ReceiptResponse `json:"receipt,omitempty"`
}

type WebhookProcessResponse struct {
	EventID   string           `json:"event_id"`
	EventType string           `json:"event_type"`
	Status    string           `json:"status"`
	Payment   *PaymentResponse `json:"payment,omitempty"`
	Receipt   *ReceiptResponse `json:"receipt,omitempty"`
}

type ReceiptResponse struct {
	ID             uuid.UUID                   `json:"id"`
	TenantID       uuid.UUID                   `json:"tenant_id"`
	ReceiptNumber  string                      `json:"receipt_number"`
	PaymentID      uuid.UUID                   `json:"payment_id"`
	StudentID      uuid.UUID                   `json:"student_id"`
	AcademicYearID uuid.UUID                   `json:"academic_year_id"`
	Status         string                      `json:"status"`
	IssueDate      string                      `json:"issue_date"`
	Currency       string                      `json:"currency"`
	AmountPaise    int64                       `json:"amount_paise"`
	PaymentMethod  string                      `json:"payment_method"`
	Metadata       map[string]any              `json:"metadata,omitempty"`
	Student        *StudentBriefResponse       `json:"student,omitempty"`
	AcademicYear   *LookupResponse             `json:"academic_year,omitempty"`
	Payment        *PaymentResponse            `json:"payment,omitempty"`
	Allocations    []PaymentAllocationResponse `json:"allocations,omitempty"`
	IssuedAt       time.Time                   `json:"issued_at"`
	CreatedAt      time.Time                   `json:"created_at"`
	UpdatedAt      time.Time                   `json:"updated_at"`
}

type ReceiptDownloadResponse struct {
	Filename    string
	ContentType string
	Bytes       []byte
}

type PaymentEventResponse struct {
	ID          uuid.UUID             `json:"id"`
	TenantID    uuid.UUID             `json:"tenant_id"`
	PaymentID   *uuid.UUID            `json:"payment_id,omitempty"`
	AttemptID   *uuid.UUID            `json:"attempt_id,omitempty"`
	ReceiptID   *uuid.UUID            `json:"receipt_id,omitempty"`
	StudentID   uuid.UUID             `json:"student_id"`
	EventType   string                `json:"event_type"`
	Status      string                `json:"status"`
	AmountPaise int64                 `json:"amount_paise"`
	Message     string                `json:"message"`
	Metadata    map[string]any        `json:"metadata,omitempty"`
	Student     *StudentBriefResponse `json:"student,omitempty"`
	OccurredAt  time.Time             `json:"occurred_at"`
	CreatedAt   time.Time             `json:"created_at"`
}
