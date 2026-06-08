package model

import (
	"time"

	"github.com/google/uuid"
)

type PaymentAttempt struct {
	ID              uuid.UUID      `json:"id"`
	TenantID        uuid.UUID      `json:"tenant_id"`
	StudentID       uuid.UUID      `json:"student_id"`
	Provider        string         `json:"provider"`
	ProviderOrderID *string        `json:"provider_order_id,omitempty"`
	IdempotencyKey  *string        `json:"idempotency_key,omitempty"`
	Status          string         `json:"status"`
	AmountPaise     int64          `json:"amount_paise"`
	Currency        string         `json:"currency"`
	CheckoutURL     string         `json:"checkout_url"`
	ExpiresAt       *time.Time     `json:"expires_at,omitempty"`
	CreatedBy       *uuid.UUID     `json:"created_by,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`

	Student     *Student            `json:"student,omitempty"`
	Allocations []PaymentAllocation `json:"allocations,omitempty"`
	Payment     *Payment            `json:"payment,omitempty"`
	Receipt     *Receipt            `json:"receipt,omitempty"`
}

type Payment struct {
	ID                 uuid.UUID      `json:"id"`
	TenantID           uuid.UUID      `json:"tenant_id"`
	AttemptID          *uuid.UUID     `json:"attempt_id,omitempty"`
	StudentID          uuid.UUID      `json:"student_id"`
	Provider           string         `json:"provider"`
	PaymentMethod      string         `json:"payment_method"`
	Status             string         `json:"status"`
	AmountPaise        int64          `json:"amount_paise"`
	AmountAppliedPaise int64          `json:"amount_applied_paise"`
	Currency           string         `json:"currency"`
	GatewayOrderID     *string        `json:"gateway_order_id,omitempty"`
	GatewayPaymentID   *string        `json:"gateway_payment_id,omitempty"`
	GatewaySignature   string         `json:"gateway_signature,omitempty"`
	ExternalReference  string         `json:"external_reference"`
	PaidAt             *time.Time     `json:"paid_at,omitempty"`
	VerifiedAt         *time.Time     `json:"verified_at,omitempty"`
	ReceivedBy         *uuid.UUID     `json:"received_by,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`

	Student     *Student            `json:"student,omitempty"`
	Attempt     *PaymentAttempt     `json:"attempt,omitempty"`
	Allocations []PaymentAllocation `json:"allocations,omitempty"`
	Receipt     *Receipt            `json:"receipt,omitempty"`
}

type PaymentAllocation struct {
	TenantID    uuid.UUID `json:"tenant_id"`
	PaymentID   uuid.UUID `json:"payment_id,omitempty"`
	AttemptID   uuid.UUID `json:"attempt_id,omitempty"`
	InvoiceID   uuid.UUID `json:"invoice_id"`
	AmountPaise int64     `json:"amount_paise"`
	CreatedAt   time.Time `json:"created_at"`
	Invoice     *Invoice  `json:"invoice,omitempty"`
}

type GatewayWebhook struct {
	ID               uuid.UUID      `json:"id"`
	TenantID         uuid.UUID      `json:"tenant_id"`
	Provider         string         `json:"provider"`
	EventID          string         `json:"event_id"`
	EventType        string         `json:"event_type"`
	ProcessingStatus string         `json:"processing_status"`
	Payload          map[string]any `json:"payload,omitempty"`
	Signature        string         `json:"signature,omitempty"`
	ErrorMessage     string         `json:"error_message,omitempty"`
	ReceivedAt       time.Time      `json:"received_at"`
	ProcessedAt      *time.Time     `json:"processed_at,omitempty"`
}

type Receipt struct {
	ID             uuid.UUID      `json:"id"`
	TenantID       uuid.UUID      `json:"tenant_id"`
	ReceiptNumber  string         `json:"receipt_number"`
	PaymentID      uuid.UUID      `json:"payment_id"`
	StudentID      uuid.UUID      `json:"student_id"`
	AcademicYearID uuid.UUID      `json:"academic_year_id"`
	BranchID       *uuid.UUID     `json:"branch_id,omitempty"`
	Status         string         `json:"status"`
	IssueDate      time.Time      `json:"issue_date"`
	Currency       string         `json:"currency"`
	AmountPaise    int64          `json:"amount_paise"`
	PaymentMethod  string         `json:"payment_method"`
	IssuedBy       *uuid.UUID     `json:"issued_by,omitempty"`
	IssuedAt       time.Time      `json:"issued_at"`
	CancelledAt    *time.Time     `json:"cancelled_at,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`

	Student      *Student            `json:"student,omitempty"`
	AcademicYear *AcademicYear       `json:"academic_year,omitempty"`
	Payment      *Payment            `json:"payment,omitempty"`
	Allocations  []PaymentAllocation `json:"allocations,omitempty"`
}

type OfflinePaymentReference struct {
	ID              uuid.UUID      `json:"id"`
	TenantID        uuid.UUID      `json:"tenant_id"`
	PaymentID       uuid.UUID      `json:"payment_id"`
	PaymentMethod   string         `json:"payment_method"`
	ReferenceNumber string         `json:"reference_number"`
	BankName        string         `json:"bank_name"`
	InstrumentDate  *time.Time     `json:"instrument_date,omitempty"`
	DepositedAt     *time.Time     `json:"deposited_at,omitempty"`
	ClearanceStatus string         `json:"clearance_status"`
	Remarks         string         `json:"remarks"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type PaymentEvent struct {
	ID          uuid.UUID      `json:"id"`
	TenantID    uuid.UUID      `json:"tenant_id"`
	PaymentID   *uuid.UUID     `json:"payment_id,omitempty"`
	AttemptID   *uuid.UUID     `json:"attempt_id,omitempty"`
	ReceiptID   *uuid.UUID     `json:"receipt_id,omitempty"`
	StudentID   uuid.UUID      `json:"student_id"`
	EventType   string         `json:"event_type"`
	Status      string         `json:"status"`
	AmountPaise int64          `json:"amount_paise"`
	Message     string         `json:"message"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	OccurredAt  time.Time      `json:"occurred_at"`
	CreatedAt   time.Time      `json:"created_at"`

	Student *Student `json:"student,omitempty"`
	Payment *Payment `json:"payment,omitempty"`
	Receipt *Receipt `json:"receipt,omitempty"`
}

type PaymentFilter struct {
	StudentID     *uuid.UUID
	Status        string
	PaymentMethod string
	Provider      string
	From          *time.Time
	To            *time.Time
	Search        string
}

type ReceiptFilter struct {
	StudentID *uuid.UUID
	Status    string
	From      *time.Time
	To        *time.Time
	Search    string
}

type PaymentEventFilter struct {
	StudentID *uuid.UUID
	EventType string
	Status    string
	From      *time.Time
	To        *time.Time
}
