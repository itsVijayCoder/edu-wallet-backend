package model

import (
	"time"

	"github.com/google/uuid"
)

type FeeHead struct {
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
	DeletedAt   *time.Time     `json:"deleted_at,omitempty"`
}

type FeeStructure struct {
	ID                        uuid.UUID          `json:"id"`
	TenantID                  uuid.UUID          `json:"tenant_id"`
	AcademicYearID            uuid.UUID          `json:"academic_year_id"`
	Name                      string             `json:"name"`
	Code                      string             `json:"code"`
	Description               string             `json:"description"`
	BillingCycle              string             `json:"billing_cycle"`
	Status                    string             `json:"status"`
	Currency                  string             `json:"currency"`
	AllowPartialPayment       bool               `json:"allow_partial_payment"`
	MinimumPartialAmountPaise int64              `json:"minimum_partial_amount_paise"`
	DueDay                    *int               `json:"due_day,omitempty"`
	Metadata                  map[string]any     `json:"metadata,omitempty"`
	CreatedAt                 time.Time          `json:"created_at"`
	UpdatedAt                 time.Time          `json:"updated_at"`
	DeletedAt                 *time.Time         `json:"deleted_at,omitempty"`
	AcademicYear              *AcademicYear      `json:"academic_year,omitempty"`
	Items                     []FeeStructureItem `json:"items,omitempty"`
}

type FeeStructureItem struct {
	ID             uuid.UUID      `json:"id"`
	TenantID       uuid.UUID      `json:"tenant_id"`
	FeeStructureID uuid.UUID      `json:"fee_structure_id"`
	FeeHeadID      uuid.UUID      `json:"fee_head_id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	AmountPaise    int64          `json:"amount_paise"`
	TaxRateBps     int            `json:"tax_rate_bps"`
	SortOrder      int            `json:"sort_order"`
	Optional       bool           `json:"optional"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      *time.Time     `json:"deleted_at,omitempty"`
	FeeHead        *FeeHead       `json:"fee_head,omitempty"`
}

type StudentFeeAssignment struct {
	ID               uuid.UUID      `json:"id"`
	TenantID         uuid.UUID      `json:"tenant_id"`
	FeeStructureID   uuid.UUID      `json:"fee_structure_id"`
	FeeStructureName string         `json:"fee_structure_name,omitempty"`
	AcademicYearID   uuid.UUID      `json:"academic_year_id"`
	AcademicYearName string         `json:"academic_year_name,omitempty"`
	AssignmentType   string         `json:"assignment_type"`
	ClassID          *uuid.UUID     `json:"class_id,omitempty"`
	ClassName        string         `json:"class_name,omitempty"`
	SectionID        *uuid.UUID     `json:"section_id,omitempty"`
	SectionName      string         `json:"section_name,omitempty"`
	StudentID        *uuid.UUID     `json:"student_id,omitempty"`
	StudentName      string         `json:"student_name,omitempty"`
	Status           string         `json:"status"`
	EffectiveFrom    time.Time      `json:"effective_from"`
	EffectiveUntil   *time.Time     `json:"effective_until,omitempty"`
	CreatedBy        *uuid.UUID     `json:"created_by,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        *time.Time     `json:"deleted_at,omitempty"`

	FeeStructure *FeeStructure `json:"fee_structure,omitempty"`
	AcademicYear *AcademicYear `json:"academic_year,omitempty"`
	Class        *Class        `json:"class,omitempty"`
	Section      *Section      `json:"section,omitempty"`
	Student      *Student      `json:"student,omitempty"`
}

type Concession struct {
	ID             uuid.UUID      `json:"id"`
	TenantID       uuid.UUID      `json:"tenant_id"`
	AcademicYearID uuid.UUID      `json:"academic_year_id"`
	StudentID      uuid.UUID      `json:"student_id"`
	FeeHeadID      *uuid.UUID     `json:"fee_head_id,omitempty"`
	Name           string         `json:"name"`
	Code           string         `json:"code"`
	ConcessionType string         `json:"concession_type"`
	AmountPaise    int64          `json:"amount_paise"`
	PercentageBps  int            `json:"percentage_bps"`
	Reason         string         `json:"reason"`
	Status         string         `json:"status"`
	StartsOn       time.Time      `json:"starts_on"`
	EndsOn         *time.Time     `json:"ends_on,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      *time.Time     `json:"deleted_at,omitempty"`
}

type LateFeeRule struct {
	ID             uuid.UUID      `json:"id"`
	TenantID       uuid.UUID      `json:"tenant_id"`
	FeeStructureID *uuid.UUID     `json:"fee_structure_id,omitempty"`
	FeeHeadID      *uuid.UUID     `json:"fee_head_id,omitempty"`
	Name           string         `json:"name"`
	RuleType       string         `json:"rule_type"`
	AmountPaise    int64          `json:"amount_paise"`
	GraceDays      int            `json:"grace_days"`
	MaxAmountPaise *int64         `json:"max_amount_paise,omitempty"`
	Status         string         `json:"status"`
	EffectiveFrom  time.Time      `json:"effective_from"`
	EffectiveUntil *time.Time     `json:"effective_until,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      *time.Time     `json:"deleted_at,omitempty"`
}

type Invoice struct {
	ID                        uuid.UUID      `json:"id"`
	TenantID                  uuid.UUID      `json:"tenant_id"`
	InvoiceNumber             string         `json:"invoice_number"`
	StudentID                 uuid.UUID      `json:"student_id"`
	AcademicYearID            uuid.UUID      `json:"academic_year_id"`
	ClassID                   uuid.UUID      `json:"class_id"`
	SectionID                 uuid.UUID      `json:"section_id"`
	FeeStructureID            *uuid.UUID     `json:"fee_structure_id,omitempty"`
	AssignmentID              *uuid.UUID     `json:"assignment_id,omitempty"`
	IssueDate                 time.Time      `json:"issue_date"`
	DueDate                   time.Time      `json:"due_date"`
	BillingPeriodStart        *time.Time     `json:"billing_period_start,omitempty"`
	BillingPeriodEnd          *time.Time     `json:"billing_period_end,omitempty"`
	GenerationKey             string         `json:"generation_key"`
	Status                    string         `json:"status"`
	Currency                  string         `json:"currency"`
	AllowPartialPayment       bool           `json:"allow_partial_payment"`
	MinimumPartialAmountPaise int64          `json:"minimum_partial_amount_paise"`
	SubtotalAmountPaise       int64          `json:"subtotal_amount_paise"`
	DiscountAmountPaise       int64          `json:"discount_amount_paise"`
	FineAmountPaise           int64          `json:"fine_amount_paise"`
	TaxAmountPaise            int64          `json:"tax_amount_paise"`
	TotalAmountPaise          int64          `json:"total_amount_paise"`
	PaidAmountPaise           int64          `json:"paid_amount_paise"`
	BalanceAmountPaise        int64          `json:"balance_amount_paise"`
	GeneratedBy               *uuid.UUID     `json:"generated_by,omitempty"`
	Metadata                  map[string]any `json:"metadata,omitempty"`
	IssuedAt                  *time.Time     `json:"issued_at,omitempty"`
	CancelledAt               *time.Time     `json:"cancelled_at,omitempty"`
	VoidedAt                  *time.Time     `json:"voided_at,omitempty"`
	CreatedAt                 time.Time      `json:"created_at"`
	UpdatedAt                 time.Time      `json:"updated_at"`

	Items        []InvoiceItem `json:"items,omitempty"`
	Student      *Student      `json:"student,omitempty"`
	AcademicYear *AcademicYear `json:"academic_year,omitempty"`
	Class        *Class        `json:"class,omitempty"`
	Section      *Section      `json:"section,omitempty"`
	FeeStructure *FeeStructure `json:"fee_structure,omitempty"`
}

type InvoiceItem struct {
	ID                  uuid.UUID      `json:"id"`
	TenantID            uuid.UUID      `json:"tenant_id"`
	InvoiceID           uuid.UUID      `json:"invoice_id"`
	FeeHeadID           uuid.UUID      `json:"fee_head_id"`
	FeeStructureItemID  *uuid.UUID     `json:"fee_structure_item_id,omitempty"`
	Description         string         `json:"description"`
	AmountPaise         int64          `json:"amount_paise"`
	DiscountAmountPaise int64          `json:"discount_amount_paise"`
	FineAmountPaise     int64          `json:"fine_amount_paise"`
	TaxAmountPaise      int64          `json:"tax_amount_paise"`
	TotalAmountPaise    int64          `json:"total_amount_paise"`
	SortOrder           int            `json:"sort_order"`
	Metadata            map[string]any `json:"metadata,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	FeeHead             *FeeHead       `json:"fee_head,omitempty"`
}

type FeeHeadFilter struct {
	Status   string
	Category string
	Search   string
}

type FeeStructureFilter struct {
	AcademicYearID *uuid.UUID
	Status         string
	BillingCycle   string
	Search         string
}

type FeeAssignmentFilter struct {
	FeeStructureID *uuid.UUID
	AcademicYearID *uuid.UUID
	AssignmentType string
	Status         string
	Search         string
}

type InvoiceFilter struct {
	StudentID      *uuid.UUID
	AcademicYearID *uuid.UUID
	ClassID        *uuid.UUID
	SectionID      *uuid.UUID
	Status         string
	DueFrom        *time.Time
	DueTo          *time.Time
	Search         string
}
