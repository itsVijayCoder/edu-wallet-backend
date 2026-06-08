package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateReminderTemplateRequest struct {
	Name     string         `json:"name" binding:"required,min=2,max=160"`
	Code     string         `json:"code" binding:"required,min=1,max=80"`
	Channel  string         `json:"channel" binding:"omitempty,oneof=email sms whatsapp in_app"`
	Subject  string         `json:"subject" binding:"omitempty,max=200"`
	Body     string         `json:"body" binding:"required,min=1"`
	Tone     string         `json:"tone" binding:"omitempty,oneof=polite formal urgent"`
	Status   string         `json:"status" binding:"omitempty,oneof=active inactive archived"`
	Metadata map[string]any `json:"metadata"`
}

type UpdateReminderTemplateRequest struct {
	Name     *string        `json:"name" binding:"omitempty,min=2,max=160"`
	Code     *string        `json:"code" binding:"omitempty,min=1,max=80"`
	Channel  *string        `json:"channel" binding:"omitempty,oneof=email sms whatsapp in_app"`
	Subject  *string        `json:"subject" binding:"omitempty,max=200"`
	Body     *string        `json:"body" binding:"omitempty,min=1"`
	Tone     *string        `json:"tone" binding:"omitempty,oneof=polite formal urgent"`
	Status   *string        `json:"status" binding:"omitempty,oneof=active inactive archived"`
	Metadata map[string]any `json:"metadata"`
}

type ReminderTemplateResponse struct {
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
}

type CreateReminderRuleRequest struct {
	TemplateID     *uuid.UUID     `json:"template_id"`
	Name           string         `json:"name" binding:"required,min=2,max=160"`
	Code           string         `json:"code" binding:"required,min=1,max=80"`
	Channel        string         `json:"channel" binding:"omitempty,oneof=email sms whatsapp in_app"`
	TriggerType    string         `json:"trigger_type" binding:"omitempty,oneof=before_due on_due after_due manual"`
	OffsetDays     int            `json:"offset_days" binding:"omitempty,min=-365,max=365"`
	TargetStatuses []string       `json:"target_statuses"`
	Status         string         `json:"status" binding:"omitempty,oneof=active inactive archived"`
	MaxAttempts    int            `json:"max_attempts" binding:"omitempty,min=1,max=10"`
	Metadata       map[string]any `json:"metadata"`
}

type UpdateReminderRuleRequest struct {
	TemplateID     *uuid.UUID     `json:"template_id"`
	ClearTemplate  bool           `json:"clear_template"`
	Name           *string        `json:"name" binding:"omitempty,min=2,max=160"`
	Code           *string        `json:"code" binding:"omitempty,min=1,max=80"`
	Channel        *string        `json:"channel" binding:"omitempty,oneof=email sms whatsapp in_app"`
	TriggerType    *string        `json:"trigger_type" binding:"omitempty,oneof=before_due on_due after_due manual"`
	OffsetDays     *int           `json:"offset_days" binding:"omitempty,min=-365,max=365"`
	TargetStatuses *[]string      `json:"target_statuses"`
	Status         *string        `json:"status" binding:"omitempty,oneof=active inactive archived"`
	MaxAttempts    *int           `json:"max_attempts" binding:"omitempty,min=1,max=10"`
	Metadata       map[string]any `json:"metadata"`
}

type ReminderRuleResponse struct {
	ID             uuid.UUID                 `json:"id"`
	TenantID       uuid.UUID                 `json:"tenant_id"`
	TemplateID     *uuid.UUID                `json:"template_id,omitempty"`
	Name           string                    `json:"name"`
	Code           string                    `json:"code"`
	Channel        string                    `json:"channel"`
	TriggerType    string                    `json:"trigger_type"`
	OffsetDays     int                       `json:"offset_days"`
	TargetStatuses []string                  `json:"target_statuses"`
	Status         string                    `json:"status"`
	MaxAttempts    int                       `json:"max_attempts"`
	Metadata       map[string]any            `json:"metadata,omitempty"`
	Template       *ReminderTemplateResponse `json:"template,omitempty"`
	CreatedAt      time.Time                 `json:"created_at"`
	UpdatedAt      time.Time                 `json:"updated_at"`
}

type SendReminderRequest struct {
	RuleID         *uuid.UUID     `json:"rule_id"`
	TemplateID     *uuid.UUID     `json:"template_id"`
	Channel        string         `json:"channel" binding:"omitempty,oneof=email sms whatsapp in_app"`
	InvoiceIDs     []uuid.UUID    `json:"invoice_ids"`
	StudentID      *uuid.UUID     `json:"student_id"`
	ClassID        *uuid.UUID     `json:"class_id"`
	SectionID      *uuid.UUID     `json:"section_id"`
	AcademicYearID *uuid.UUID     `json:"academic_year_id"`
	DueOnOrBefore  string         `json:"due_on_or_before"`
	Subject        string         `json:"subject" binding:"omitempty,max=200"`
	Message        string         `json:"message"`
	ProcessNow     *bool          `json:"process_now"`
	Metadata       map[string]any `json:"metadata"`
}

type SendReminderResponse struct {
	QueuedCount  int                   `json:"queued_count"`
	SentCount    int                   `json:"sent_count"`
	FailedCount  int                   `json:"failed_count"`
	SkippedCount int                   `json:"skipped_count"`
	ReminderLogs []ReminderLogResponse `json:"reminder_logs"`
}

type ReminderLogResponse struct {
	ID                uuid.UUID             `json:"id"`
	TenantID          uuid.UUID             `json:"tenant_id"`
	RuleID            *uuid.UUID            `json:"rule_id,omitempty"`
	TemplateID        *uuid.UUID            `json:"template_id,omitempty"`
	JobID             *uuid.UUID            `json:"job_id,omitempty"`
	InvoiceID         *uuid.UUID            `json:"invoice_id,omitempty"`
	StudentID         uuid.UUID             `json:"student_id"`
	GuardianID        *uuid.UUID            `json:"guardian_id,omitempty"`
	Channel           string                `json:"channel"`
	Recipient         string                `json:"recipient"`
	Subject           string                `json:"subject"`
	Message           string                `json:"message"`
	Status            string                `json:"status"`
	Provider          string                `json:"provider,omitempty"`
	ProviderMessageID string                `json:"provider_message_id,omitempty"`
	ErrorMessage      string                `json:"error_message,omitempty"`
	ScheduledFor      time.Time             `json:"scheduled_for"`
	AttemptedAt       *time.Time            `json:"attempted_at,omitempty"`
	SentAt            *time.Time            `json:"sent_at,omitempty"`
	AttemptCount      int                   `json:"attempt_count"`
	Metadata          map[string]any        `json:"metadata,omitempty"`
	Student           *StudentBriefResponse `json:"student,omitempty"`
	GuardianName      string                `json:"guardian_name,omitempty"`
	InvoiceNumber     string                `json:"invoice_number,omitempty"`
	CreatedAt         time.Time             `json:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at"`
}

type DashboardResponse struct {
	Currency               string                         `json:"currency"`
	TotalStudents          int64                          `json:"total_students"`
	ActiveStudents         int64                          `json:"active_students"`
	TodayCollectionPaise   int64                          `json:"today_collection_paise"`
	MonthCollectionPaise   int64                          `json:"month_collection_paise"`
	TotalDuePaise          int64                          `json:"total_due_paise"`
	OverduePaise           int64                          `json:"overdue_paise"`
	DefaulterCount         int64                          `json:"defaulter_count"`
	UnpaidInvoiceCount     int64                          `json:"unpaid_invoice_count"`
	PaymentMethodBreakdown []PaymentMethodSummaryResponse `json:"payment_method_breakdown"`
	RecentPaymentEvents    []PaymentEventResponse         `json:"recent_payment_events"`
}

type PaymentMethodSummaryResponse struct {
	PaymentMethod string `json:"payment_method"`
	PaymentCount  int64  `json:"payment_count"`
	AmountPaise   int64  `json:"amount_paise"`
}

type CollectionReportRowResponse struct {
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

type DefaulterReportRowResponse struct {
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
	OldestDueDate      string     `json:"oldest_due_date"`
	LastReminderAt     *time.Time `json:"last_reminder_at,omitempty"`
	LastReminderStatus string     `json:"last_reminder_status,omitempty"`
}

type DueReportRowResponse struct {
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

type FeeHeadCollectionRowResponse struct {
	FeeHeadID      uuid.UUID `json:"fee_head_id"`
	FeeHeadName    string    `json:"fee_head_name"`
	FeeHeadCode    string    `json:"fee_head_code"`
	Category       string    `json:"category"`
	CollectedPaise int64     `json:"collected_paise"`
	InvoiceCount   int64     `json:"invoice_count"`
	PaymentCount   int64     `json:"payment_count"`
}

type PaymentMethodReportRowResponse struct {
	PaymentMethod string `json:"payment_method"`
	Provider      string `json:"provider"`
	PaymentCount  int64  `json:"payment_count"`
	AmountPaise   int64  `json:"amount_paise"`
}

type CreateExportRequest struct {
	ExportType    string         `json:"export_type" binding:"required,oneof=collections defaulters dues payment_methods fee_heads offline_payments receipt_register"`
	Format        string         `json:"format" binding:"omitempty,oneof=csv"`
	From          string         `json:"from"`
	To            string         `json:"to"`
	AsOf          string         `json:"as_of"`
	StudentID     *uuid.UUID     `json:"student_id"`
	ClassID       *uuid.UUID     `json:"class_id"`
	SectionID     *uuid.UUID     `json:"section_id"`
	PaymentMethod string         `json:"payment_method"`
	Provider      string         `json:"provider"`
	Metadata      map[string]any `json:"metadata"`
}

type ExportJobResponse struct {
	ID           uuid.UUID      `json:"id"`
	TenantID     uuid.UUID      `json:"tenant_id"`
	ExportType   string         `json:"export_type"`
	Status       string         `json:"status"`
	Format       string         `json:"format"`
	Params       map[string]any `json:"params,omitempty"`
	FileName     string         `json:"file_name"`
	ContentType  string         `json:"content_type"`
	RowCount     int            `json:"row_count"`
	RequestedBy  *uuid.UUID     `json:"requested_by,omitempty"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type ExportDownloadResponse struct {
	Filename    string
	ContentType string
	Bytes       []byte
}
