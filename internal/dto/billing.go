package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateFeeHeadRequest struct {
	Name        string         `json:"name" binding:"required,min=2,max=160"`
	Code        string         `json:"code" binding:"required,min=1,max=60"`
	Description string         `json:"description"`
	Category    string         `json:"category" binding:"omitempty,oneof=admission tuition term exam transport hostel lab library uniform_books fine activity sports development certificate mess id_card miscellaneous custom"`
	Status      string         `json:"status" binding:"omitempty,oneof=active inactive"`
	Taxable     bool           `json:"taxable"`
	TaxRateBps  int            `json:"tax_rate_bps" binding:"omitempty,min=0,max=10000"`
	Metadata    map[string]any `json:"metadata"`
}

type UpdateFeeHeadRequest struct {
	Name        *string        `json:"name" binding:"omitempty,min=2,max=160"`
	Code        *string        `json:"code" binding:"omitempty,min=1,max=60"`
	Description *string        `json:"description"`
	Category    *string        `json:"category" binding:"omitempty,oneof=admission tuition term exam transport hostel lab library uniform_books fine activity sports development certificate mess id_card miscellaneous custom"`
	Status      *string        `json:"status" binding:"omitempty,oneof=active inactive"`
	Taxable     *bool          `json:"taxable"`
	TaxRateBps  *int           `json:"tax_rate_bps" binding:"omitempty,min=0,max=10000"`
	Metadata    map[string]any `json:"metadata"`
}

type FeeHeadResponse struct {
	ID          uuid.UUID      `json:"id"`
	TenantID    uuid.UUID      `json:"tenant_id"`
	Name        string         `json:"name"`
	Code        string         `json:"code"`
	Description string         `json:"description"`
	Category    string         `json:"category"`
	Status      string         `json:"status"`
	Taxable     bool           `json:"taxable"`
	TaxRateBps  int            `json:"tax_rate_bps"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type CreateFeeStructureItemRequest struct {
	FeeHeadID   uuid.UUID      `json:"fee_head_id" binding:"required"`
	Name        string         `json:"name" binding:"omitempty,max=160"`
	Description string         `json:"description"`
	AmountPaise int64          `json:"amount_paise" binding:"required,min=0"`
	TaxRateBps  *int           `json:"tax_rate_bps" binding:"omitempty,min=0,max=10000"`
	SortOrder   int            `json:"sort_order"`
	Optional    bool           `json:"optional"`
	Metadata    map[string]any `json:"metadata"`
}

type CreateFeeStructureRequest struct {
	AcademicYearID            uuid.UUID                       `json:"academic_year_id" binding:"required"`
	Name                      string                          `json:"name" binding:"required,min=2,max=160"`
	Code                      string                          `json:"code" binding:"required,min=1,max=60"`
	Description               string                          `json:"description"`
	BillingCycle              string                          `json:"billing_cycle" binding:"omitempty,oneof=one_time monthly quarterly term yearly custom"`
	Status                    string                          `json:"status" binding:"omitempty,oneof=draft active inactive archived"`
	AllowPartialPayment       bool                            `json:"allow_partial_payment"`
	MinimumPartialAmountPaise int64                           `json:"minimum_partial_amount_paise" binding:"omitempty,min=0"`
	DueDay                    *int                            `json:"due_day" binding:"omitempty,min=1,max=31"`
	Metadata                  map[string]any                  `json:"metadata"`
	Items                     []CreateFeeStructureItemRequest `json:"items" binding:"required,min=1,dive"`
}

type UpdateFeeStructureRequest struct {
	AcademicYearID            *uuid.UUID                       `json:"academic_year_id"`
	Name                      *string                          `json:"name" binding:"omitempty,min=2,max=160"`
	Code                      *string                          `json:"code" binding:"omitempty,min=1,max=60"`
	Description               *string                          `json:"description"`
	BillingCycle              *string                          `json:"billing_cycle" binding:"omitempty,oneof=one_time monthly quarterly term yearly custom"`
	Status                    *string                          `json:"status" binding:"omitempty,oneof=draft active inactive archived"`
	AllowPartialPayment       *bool                            `json:"allow_partial_payment"`
	MinimumPartialAmountPaise *int64                           `json:"minimum_partial_amount_paise" binding:"omitempty,min=0"`
	DueDay                    *int                             `json:"due_day" binding:"omitempty,min=1,max=31"`
	Metadata                  map[string]any                   `json:"metadata"`
	Items                     *[]CreateFeeStructureItemRequest `json:"items" binding:"omitempty,dive"`
}

type FeeStructureResponse struct {
	ID                        uuid.UUID                  `json:"id"`
	TenantID                  uuid.UUID                  `json:"tenant_id"`
	AcademicYearID            uuid.UUID                  `json:"academic_year_id"`
	Name                      string                     `json:"name"`
	Code                      string                     `json:"code"`
	Description               string                     `json:"description"`
	BillingCycle              string                     `json:"billing_cycle"`
	Status                    string                     `json:"status"`
	Currency                  string                     `json:"currency"`
	AllowPartialPayment       bool                       `json:"allow_partial_payment"`
	MinimumPartialAmountPaise int64                      `json:"minimum_partial_amount_paise"`
	DueDay                    *int                       `json:"due_day,omitempty"`
	Metadata                  map[string]any             `json:"metadata,omitempty"`
	AcademicYear              *LookupResponse            `json:"academic_year,omitempty"`
	Items                     []FeeStructureItemResponse `json:"items,omitempty"`
	CreatedAt                 time.Time                  `json:"created_at"`
	UpdatedAt                 time.Time                  `json:"updated_at"`
}

type FeeStructureItemResponse struct {
	ID             uuid.UUID        `json:"id"`
	FeeStructureID uuid.UUID        `json:"fee_structure_id"`
	FeeHeadID      uuid.UUID        `json:"fee_head_id"`
	Name           string           `json:"name"`
	Description    string           `json:"description"`
	AmountPaise    int64            `json:"amount_paise"`
	TaxRateBps     int              `json:"tax_rate_bps"`
	SortOrder      int              `json:"sort_order"`
	Optional       bool             `json:"optional"`
	Metadata       map[string]any   `json:"metadata,omitempty"`
	FeeHead        *FeeHeadResponse `json:"fee_head,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type CreateFeeAssignmentRequest struct {
	FeeStructureID uuid.UUID      `json:"fee_structure_id" binding:"required"`
	AssignmentType string         `json:"assignment_type" binding:"required,oneof=class section student"`
	AcademicYearID *uuid.UUID     `json:"academic_year_id"`
	ClassID        *uuid.UUID     `json:"class_id"`
	SectionID      *uuid.UUID     `json:"section_id"`
	StudentID      *uuid.UUID     `json:"student_id"`
	EffectiveFrom  string         `json:"effective_from"`
	EffectiveUntil *string        `json:"effective_until"`
	Status         string         `json:"status" binding:"omitempty,oneof=active inactive cancelled"`
	Metadata       map[string]any `json:"metadata"`
}

type FeeAssignmentResponse struct {
	ID             uuid.UUID             `json:"id"`
	TenantID       uuid.UUID             `json:"tenant_id"`
	FeeStructureID uuid.UUID             `json:"fee_structure_id"`
	AcademicYearID uuid.UUID             `json:"academic_year_id"`
	AssignmentType string                `json:"assignment_type"`
	ClassID        *uuid.UUID            `json:"class_id,omitempty"`
	SectionID      *uuid.UUID            `json:"section_id,omitempty"`
	StudentID      *uuid.UUID            `json:"student_id,omitempty"`
	Status         string                `json:"status"`
	EffectiveFrom  string                `json:"effective_from"`
	EffectiveUntil *string               `json:"effective_until,omitempty"`
	Metadata       map[string]any        `json:"metadata,omitempty"`
	FeeStructure   *FeeStructureResponse `json:"fee_structure,omitempty"`
	AcademicYear   *LookupResponse       `json:"academic_year,omitempty"`
	Class          *LookupResponse       `json:"class,omitempty"`
	Section        *LookupResponse       `json:"section,omitempty"`
	Student        *StudentBriefResponse `json:"student,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

type StudentBriefResponse struct {
	ID              uuid.UUID `json:"id"`
	AdmissionNumber string    `json:"admission_number"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	Status          string    `json:"status"`
}

type GenerateInvoicesRequest struct {
	AssignmentID       uuid.UUID      `json:"assignment_id" binding:"required"`
	IssueDate          string         `json:"issue_date"`
	DueDate            string         `json:"due_date"`
	BillingPeriodStart *string        `json:"billing_period_start"`
	BillingPeriodEnd   *string        `json:"billing_period_end"`
	StudentIDs         []uuid.UUID    `json:"student_ids"`
	Metadata           map[string]any `json:"metadata"`
}

type GenerateInvoicesResponse struct {
	AssignmentID   uuid.UUID         `json:"assignment_id"`
	GeneratedCount int               `json:"generated_count"`
	SkippedCount   int               `json:"skipped_count"`
	Invoices       []InvoiceResponse `json:"invoices"`
}

type InvoiceResponse struct {
	ID                        uuid.UUID             `json:"id"`
	TenantID                  uuid.UUID             `json:"tenant_id"`
	InvoiceNumber             string                `json:"invoice_number"`
	StudentID                 uuid.UUID             `json:"student_id"`
	AcademicYearID            uuid.UUID             `json:"academic_year_id"`
	ClassID                   uuid.UUID             `json:"class_id"`
	SectionID                 uuid.UUID             `json:"section_id"`
	FeeStructureID            *uuid.UUID            `json:"fee_structure_id,omitempty"`
	AssignmentID              *uuid.UUID            `json:"assignment_id,omitempty"`
	IssueDate                 string                `json:"issue_date"`
	DueDate                   string                `json:"due_date"`
	BillingPeriodStart        *string               `json:"billing_period_start,omitempty"`
	BillingPeriodEnd          *string               `json:"billing_period_end,omitempty"`
	Status                    string                `json:"status"`
	Currency                  string                `json:"currency"`
	AllowPartialPayment       bool                  `json:"allow_partial_payment"`
	MinimumPartialAmountPaise int64                 `json:"minimum_partial_amount_paise"`
	SubtotalAmountPaise       int64                 `json:"subtotal_amount_paise"`
	DiscountAmountPaise       int64                 `json:"discount_amount_paise"`
	FineAmountPaise           int64                 `json:"fine_amount_paise"`
	TaxAmountPaise            int64                 `json:"tax_amount_paise"`
	TotalAmountPaise          int64                 `json:"total_amount_paise"`
	PaidAmountPaise           int64                 `json:"paid_amount_paise"`
	BalanceAmountPaise        int64                 `json:"balance_amount_paise"`
	Metadata                  map[string]any        `json:"metadata,omitempty"`
	Student                   *StudentBriefResponse `json:"student,omitempty"`
	AcademicYear              *LookupResponse       `json:"academic_year,omitempty"`
	Class                     *LookupResponse       `json:"class,omitempty"`
	Section                   *LookupResponse       `json:"section,omitempty"`
	FeeStructure              *LookupResponse       `json:"fee_structure,omitempty"`
	Items                     []InvoiceItemResponse `json:"items,omitempty"`
	CreatedAt                 time.Time             `json:"created_at"`
	UpdatedAt                 time.Time             `json:"updated_at"`
}

type InvoiceItemResponse struct {
	ID                  uuid.UUID        `json:"id"`
	InvoiceID           uuid.UUID        `json:"invoice_id"`
	FeeHeadID           uuid.UUID        `json:"fee_head_id"`
	FeeStructureItemID  *uuid.UUID       `json:"fee_structure_item_id,omitempty"`
	Description         string           `json:"description"`
	AmountPaise         int64            `json:"amount_paise"`
	DiscountAmountPaise int64            `json:"discount_amount_paise"`
	FineAmountPaise     int64            `json:"fine_amount_paise"`
	TaxAmountPaise      int64            `json:"tax_amount_paise"`
	TotalAmountPaise    int64            `json:"total_amount_paise"`
	SortOrder           int              `json:"sort_order"`
	Metadata            map[string]any   `json:"metadata,omitempty"`
	FeeHead             *FeeHeadResponse `json:"fee_head,omitempty"`
	CreatedAt           time.Time        `json:"created_at"`
}

type StudentLedgerEntryResponse struct {
	EntryType         string    `json:"entry_type"`
	EntryID           uuid.UUID `json:"entry_id"`
	ReferenceNumber   string    `json:"reference_number"`
	Description       string    `json:"description"`
	EntryDate         string    `json:"entry_date"`
	DueDate           *string   `json:"due_date,omitempty"`
	DebitAmountPaise  int64     `json:"debit_amount_paise"`
	CreditAmountPaise int64     `json:"credit_amount_paise"`
	BalanceAfterPaise int64     `json:"balance_after_paise"`
	Status            string    `json:"status"`
}

type StudentLedgerResponse struct {
	Student             StudentBriefResponse         `json:"student"`
	Currency            string                       `json:"currency"`
	OpeningBalancePaise int64                        `json:"opening_balance_paise"`
	TotalBilledPaise    int64                        `json:"total_billed_paise"`
	TotalPaidPaise      int64                        `json:"total_paid_paise"`
	BalancePaise        int64                        `json:"balance_paise"`
	Entries             []StudentLedgerEntryResponse `json:"entries"`
}

type ParentDuesResponse struct {
	Student             StudentBriefResponse `json:"student"`
	Currency            string               `json:"currency"`
	TotalDuePaise       int64                `json:"total_due_paise"`
	OverduePaise        int64                `json:"overdue_paise"`
	AllowPartial        bool                 `json:"allow_partial"`
	MinimumPayablePaise int64                `json:"minimum_payable_paise"`
	Invoices            []InvoiceResponse    `json:"invoices"`
}
