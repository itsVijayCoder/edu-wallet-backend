package model

import (
	"time"

	"github.com/google/uuid"
)

type ReminderTemplate struct {
	ID        uuid.UUID      `json:"id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	Name      string         `json:"name"`
	Code      string         `json:"code"`
	Channel   string         `json:"channel"`
	Subject   string         `json:"subject"`
	Body      string         `json:"body"`
	Tone      string         `json:"tone"`
	Status    string         `json:"status"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt *time.Time     `json:"deleted_at,omitempty"`
}

type ReminderRule struct {
	ID             uuid.UUID      `json:"id"`
	TenantID       uuid.UUID      `json:"tenant_id"`
	TemplateID     *uuid.UUID     `json:"template_id,omitempty"`
	Name           string         `json:"name"`
	Code           string         `json:"code"`
	Channel        string         `json:"channel"`
	TriggerType    string         `json:"trigger_type"`
	OffsetDays     int            `json:"offset_days"`
	TargetStatuses []string       `json:"target_statuses"`
	Status         string         `json:"status"`
	MaxAttempts    int            `json:"max_attempts"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      *time.Time     `json:"deleted_at,omitempty"`

	Template *ReminderTemplate `json:"template,omitempty"`
}

type Job struct {
	ID             uuid.UUID      `json:"id"`
	TenantID       uuid.UUID      `json:"tenant_id"`
	JobType        string         `json:"job_type"`
	Status         string         `json:"status"`
	Priority       int            `json:"priority"`
	RunAt          time.Time      `json:"run_at"`
	Attempts       int            `json:"attempts"`
	MaxAttempts    int            `json:"max_attempts"`
	LockedAt       *time.Time     `json:"locked_at,omitempty"`
	LockedBy       string         `json:"locked_by"`
	IdempotencyKey *string        `json:"idempotency_key,omitempty"`
	Payload        map[string]any `json:"payload,omitempty"`
	LastError      string         `json:"last_error"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type ReminderLog struct {
	ID                uuid.UUID      `json:"id"`
	TenantID          uuid.UUID      `json:"tenant_id"`
	RuleID            *uuid.UUID     `json:"rule_id,omitempty"`
	TemplateID        *uuid.UUID     `json:"template_id,omitempty"`
	JobID             *uuid.UUID     `json:"job_id,omitempty"`
	InvoiceID         *uuid.UUID     `json:"invoice_id,omitempty"`
	StudentID         uuid.UUID      `json:"student_id"`
	GuardianID        *uuid.UUID     `json:"guardian_id,omitempty"`
	Channel           string         `json:"channel"`
	Recipient         string         `json:"recipient"`
	Subject           string         `json:"subject"`
	Message           string         `json:"message"`
	Status            string         `json:"status"`
	Provider          string         `json:"provider"`
	ProviderMessageID string         `json:"provider_message_id"`
	ProviderResponse  map[string]any `json:"provider_response,omitempty"`
	ErrorMessage      string         `json:"error_message"`
	ScheduledFor      time.Time      `json:"scheduled_for"`
	AttemptedAt       *time.Time     `json:"attempted_at,omitempty"`
	SentAt            *time.Time     `json:"sent_at,omitempty"`
	AttemptCount      int            `json:"attempt_count"`
	CreatedBy         *uuid.UUID     `json:"created_by,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`

	Student  *Student          `json:"student,omitempty"`
	Guardian *Guardian         `json:"guardian,omitempty"`
	Invoice  *Invoice          `json:"invoice,omitempty"`
	Rule     *ReminderRule     `json:"rule,omitempty"`
	Template *ReminderTemplate `json:"template,omitempty"`
}

type NotificationLog struct {
	ID                uuid.UUID      `json:"id"`
	TenantID          uuid.UUID      `json:"tenant_id"`
	ReminderLogID     *uuid.UUID     `json:"reminder_log_id,omitempty"`
	Channel           string         `json:"channel"`
	Recipient         string         `json:"recipient"`
	Provider          string         `json:"provider"`
	Status            string         `json:"status"`
	ProviderMessageID string         `json:"provider_message_id"`
	ProviderResponse  map[string]any `json:"provider_response,omitempty"`
	ErrorMessage      string         `json:"error_message"`
	AttemptedAt       time.Time      `json:"attempted_at"`
	CreatedAt         time.Time      `json:"created_at"`
}

type ExportJob struct {
	ID           uuid.UUID      `json:"id"`
	TenantID     uuid.UUID      `json:"tenant_id"`
	ExportType   string         `json:"export_type"`
	Status       string         `json:"status"`
	Format       string         `json:"format"`
	Params       map[string]any `json:"params,omitempty"`
	FileName     string         `json:"file_name"`
	ContentType  string         `json:"content_type"`
	Content      []byte         `json:"-"`
	RowCount     int            `json:"row_count"`
	RequestedBy  *uuid.UUID     `json:"requested_by,omitempty"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	ErrorMessage string         `json:"error_message"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type ReminderTemplateFilter struct {
	Channel string
	Status  string
	Search  string
}

type ReminderRuleFilter struct {
	Channel     string
	TriggerType string
	Status      string
	Search      string
}

type ReminderLogFilter struct {
	StudentID *uuid.UUID
	InvoiceID *uuid.UUID
	Channel   string
	Status    string
	From      *time.Time
	To        *time.Time
}

type ReminderCandidateFilter struct {
	InvoiceIDs     []uuid.UUID
	StudentID      *uuid.UUID
	ClassID        *uuid.UUID
	SectionID      *uuid.UUID
	AcademicYearID *uuid.UUID
	DueOnOrBefore  *time.Time
	Statuses       []string
}

type ExportJobFilter struct {
	ExportType string
	Status     string
	From       *time.Time
	To         *time.Time
}

type ReportFilter struct {
	From           *time.Time
	To             *time.Time
	AsOf           *time.Time
	StudentID      *uuid.UUID
	ClassID        *uuid.UUID
	SectionID      *uuid.UUID
	AcademicYearID *uuid.UUID
	PaymentMethod  string
	Provider       string
}

type DashboardSummary struct {
	Currency               string                 `json:"currency"`
	TotalStudents          int64                  `json:"total_students"`
	ActiveStudents         int64                  `json:"active_students"`
	TodayCollectionPaise   int64                  `json:"today_collection_paise"`
	MonthCollectionPaise   int64                  `json:"month_collection_paise"`
	TotalDuePaise          int64                  `json:"total_due_paise"`
	OverduePaise           int64                  `json:"overdue_paise"`
	DefaulterCount         int64                  `json:"defaulter_count"`
	UnpaidInvoiceCount     int64                  `json:"unpaid_invoice_count"`
	PaymentMethodBreakdown []PaymentMethodSummary `json:"payment_method_breakdown"`
	RecentPaymentEvents    []PaymentEvent         `json:"recent_payment_events"`
}

type PaymentMethodSummary struct {
	PaymentMethod string `json:"payment_method"`
	PaymentCount  int64  `json:"payment_count"`
	AmountPaise   int64  `json:"amount_paise"`
}

type CollectionReportRow struct {
	PaymentID          uuid.UUID  `json:"payment_id"`
	ReceiptID          *uuid.UUID `json:"receipt_id,omitempty"`
	ReceiptNumber      string     `json:"receipt_number"`
	StudentID          uuid.UUID  `json:"student_id"`
	AdmissionNumber    string     `json:"admission_number"`
	StudentName        string     `json:"student_name"`
	ClassName          string     `json:"class_name"`
	SectionName        string     `json:"section_name"`
	PaymentMethod      string     `json:"payment_method"`
	Provider           string     `json:"provider"`
	AmountPaise        int64      `json:"amount_paise"`
	AmountAppliedPaise int64      `json:"amount_applied_paise"`
	PaidAt             time.Time  `json:"paid_at"`
	ReceiptIssuedAt    *time.Time `json:"receipt_issued_at,omitempty"`
}

type DefaulterReportRow struct {
	StudentID          uuid.UUID  `json:"student_id"`
	AdmissionNumber    string     `json:"admission_number"`
	StudentName        string     `json:"student_name"`
	ClassID            uuid.UUID  `json:"class_id"`
	ClassName          string     `json:"class_name"`
	SectionID          uuid.UUID  `json:"section_id"`
	SectionName        string     `json:"section_name"`
	GuardianName       string     `json:"guardian_name"`
	GuardianPhone      string     `json:"guardian_phone"`
	GuardianEmail      string     `json:"guardian_email"`
	InvoiceCount       int64      `json:"invoice_count"`
	TotalDuePaise      int64      `json:"total_due_paise"`
	OverduePaise       int64      `json:"overdue_paise"`
	OldestDueDate      time.Time  `json:"oldest_due_date"`
	LastReminderAt     *time.Time `json:"last_reminder_at,omitempty"`
	LastReminderStatus string     `json:"last_reminder_status"`
}

type DueReportRow struct {
	ClassID          uuid.UUID `json:"class_id"`
	ClassName        string    `json:"class_name"`
	SectionID        uuid.UUID `json:"section_id"`
	SectionName      string    `json:"section_name"`
	StudentCount     int64     `json:"student_count"`
	InvoiceCount     int64     `json:"invoice_count"`
	TotalBilledPaise int64     `json:"total_billed_paise"`
	TotalPaidPaise   int64     `json:"total_paid_paise"`
	TotalDuePaise    int64     `json:"total_due_paise"`
	OverduePaise     int64     `json:"overdue_paise"`
}

type FeeHeadCollectionRow struct {
	FeeHeadID      uuid.UUID `json:"fee_head_id"`
	FeeHeadName    string    `json:"fee_head_name"`
	FeeHeadCode    string    `json:"fee_head_code"`
	Category       string    `json:"category"`
	CollectedPaise int64     `json:"collected_paise"`
	InvoiceCount   int64     `json:"invoice_count"`
	PaymentCount   int64     `json:"payment_count"`
}

type PaymentMethodReportRow struct {
	PaymentMethod string `json:"payment_method"`
	Provider      string `json:"provider"`
	PaymentCount  int64  `json:"payment_count"`
	AmountPaise   int64  `json:"amount_paise"`
}

type ReminderCandidate struct {
	InvoiceID          uuid.UUID  `json:"invoice_id"`
	InvoiceNumber      string     `json:"invoice_number"`
	StudentID          uuid.UUID  `json:"student_id"`
	AdmissionNumber    string     `json:"admission_number"`
	StudentFirstName   string     `json:"student_first_name"`
	StudentLastName    string     `json:"student_last_name"`
	StudentEmail       *string    `json:"student_email,omitempty"`
	StudentPhone       *string    `json:"student_phone,omitempty"`
	GuardianID         *uuid.UUID `json:"guardian_id,omitempty"`
	GuardianName       string     `json:"guardian_name"`
	GuardianEmail      *string    `json:"guardian_email,omitempty"`
	GuardianPhone      *string    `json:"guardian_phone,omitempty"`
	GuardianWhatsApp   *string    `json:"guardian_whatsapp,omitempty"`
	ClassName          string     `json:"class_name"`
	SectionName        string     `json:"section_name"`
	DueDate            time.Time  `json:"due_date"`
	BalanceAmountPaise int64      `json:"balance_amount_paise"`
	Currency           string     `json:"currency"`
}
