package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
)

// RoleRepository defines data-access operations for roles.
type RoleRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Role, error)
	GetBySlug(ctx context.Context, slug string) (*model.Role, error)
	List(ctx context.Context) ([]model.Role, error)
}

// UserRepository defines data-access operations for users.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	List(ctx context.Context, params model.PaginationParams) (*model.PaginatedResult[model.User], error)
	Update(ctx context.Context, user *model.User) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	AssignRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error
	GetRoles(ctx context.Context, userID uuid.UUID) ([]model.Role, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

// SessionRepository defines data-access operations for sessions / refresh tokens.
type SessionRepository interface {
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
}

// TenantRepository defines tenant and branch data-access operations.
type TenantRepository interface {
	Create(ctx context.Context, tenant *model.Tenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*model.Tenant, error)
	List(ctx context.Context, params model.PaginationParams) (*model.PaginatedResult[model.Tenant], error)
	Update(ctx context.Context, tenant *model.Tenant) error
	CreateBranch(ctx context.Context, branch *model.TenantBranch) error
	ListBranches(ctx context.Context, tenantID uuid.UUID) ([]model.TenantBranch, error)
}

// TenantMembershipRepository defines user-to-tenant access operations.
type TenantMembershipRepository interface {
	CreateMembership(ctx context.Context, membership *model.TenantMembership) error
	GetByUserAndTenant(ctx context.Context, userID, tenantID uuid.UUID) (*model.TenantMembership, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.TenantMembership, error)
	ListPermissionsByRole(ctx context.Context, roleID uuid.UUID) ([]model.Permission, error)
}

// AuditRepository writes immutable audit events.
type AuditRepository interface {
	Create(ctx context.Context, entry *model.AuditLog) error
}

type AcademicRepositoryFactory func(db database.DBTX) AcademicRepository

type BillingRepositoryFactory func(db database.DBTX) BillingRepository

type PaymentRepositoryFactory func(db database.DBTX) PaymentRepository

type OperationsRepositoryFactory func(db database.DBTX) OperationsRepository

// AcademicRepository defines tenant-scoped academic setup, student, guardian, and import operations.
type AcademicRepository interface {
	CreateAcademicYear(ctx context.Context, academicYear *model.AcademicYear) error
	GetAcademicYear(ctx context.Context, tenantID, id uuid.UUID) (*model.AcademicYear, error)
	GetAcademicYearByCode(ctx context.Context, tenantID uuid.UUID, code string) (*model.AcademicYear, error)
	ListAcademicYears(ctx context.Context, tenantID uuid.UUID, filter model.AcademicYearFilter, params model.PaginationParams) (*model.PaginatedResult[model.AcademicYear], error)
	UpdateAcademicYear(ctx context.Context, academicYear *model.AcademicYear) error
	SoftDeleteAcademicYear(ctx context.Context, tenantID, id uuid.UUID) error

	CreateClass(ctx context.Context, class *model.Class) error
	GetClass(ctx context.Context, tenantID, id uuid.UUID) (*model.Class, error)
	GetClassByCode(ctx context.Context, tenantID uuid.UUID, code string) (*model.Class, error)
	ListClasses(ctx context.Context, tenantID uuid.UUID, filter model.ClassFilter, params model.PaginationParams) (*model.PaginatedResult[model.Class], error)
	UpdateClass(ctx context.Context, class *model.Class) error
	SoftDeleteClass(ctx context.Context, tenantID, id uuid.UUID) error

	CreateSection(ctx context.Context, section *model.Section) error
	GetSection(ctx context.Context, tenantID, id uuid.UUID) (*model.Section, error)
	GetSectionByCode(ctx context.Context, tenantID, academicYearID, classID uuid.UUID, code string) (*model.Section, error)
	ListSections(ctx context.Context, tenantID uuid.UUID, filter model.SectionFilter, params model.PaginationParams) (*model.PaginatedResult[model.Section], error)
	UpdateSection(ctx context.Context, section *model.Section) error
	SoftDeleteSection(ctx context.Context, tenantID, id uuid.UUID) error

	CreateStudent(ctx context.Context, student *model.Student) error
	GetStudent(ctx context.Context, tenantID, id uuid.UUID) (*model.Student, error)
	GetStudentByAdmissionNumber(ctx context.Context, tenantID uuid.UUID, admissionNumber string) (*model.Student, error)
	ListStudents(ctx context.Context, tenantID uuid.UUID, filter model.StudentFilter, params model.PaginationParams) (*model.PaginatedResult[model.Student], error)
	UpdateStudent(ctx context.Context, student *model.Student) error
	SoftDeleteStudent(ctx context.Context, tenantID, id uuid.UUID) error

	CreateGuardian(ctx context.Context, guardian *model.Guardian) error
	GetGuardian(ctx context.Context, tenantID, id uuid.UUID) (*model.Guardian, error)
	FindGuardianByContact(ctx context.Context, tenantID uuid.UUID, email, phone *string) (*model.Guardian, error)
	ListGuardians(ctx context.Context, tenantID uuid.UUID, filter model.GuardianFilter, params model.PaginationParams) (*model.PaginatedResult[model.Guardian], error)
	UpdateGuardian(ctx context.Context, guardian *model.Guardian) error
	SoftDeleteGuardian(ctx context.Context, tenantID, id uuid.UUID) error

	SetStudentGuardians(ctx context.Context, tenantID, studentID uuid.UUID, links []model.StudentGuardian) error
	LinkStudentGuardian(ctx context.Context, link *model.StudentGuardian) error
	UnlinkStudentGuardian(ctx context.Context, tenantID, studentID, guardianID uuid.UUID) error
	ListStudentGuardians(ctx context.Context, tenantID, studentID uuid.UUID) ([]model.StudentGuardian, error)

	CreateImport(ctx context.Context, imp *model.Import) error
	GetImport(ctx context.Context, tenantID, id uuid.UUID) (*model.Import, error)
	ListImports(ctx context.Context, tenantID uuid.UUID, filter model.ImportFilter, params model.PaginationParams) (*model.PaginatedResult[model.Import], error)
	UpdateImport(ctx context.Context, imp *model.Import) error
	CreateImportErrors(ctx context.Context, errors []model.ImportError) error
	ListImportErrors(ctx context.Context, tenantID, importID uuid.UUID) ([]model.ImportError, error)
}

// BillingRepository defines tenant-scoped fee setup, invoice generation, and ledger reads.
type BillingRepository interface {
	CreateFeeHead(ctx context.Context, feeHead *model.FeeHead) error
	GetFeeHead(ctx context.Context, tenantID, id uuid.UUID) (*model.FeeHead, error)
	GetFeeHeadByCode(ctx context.Context, tenantID uuid.UUID, code string) (*model.FeeHead, error)
	ListFeeHeads(ctx context.Context, tenantID uuid.UUID, filter model.FeeHeadFilter, params model.PaginationParams) (*model.PaginatedResult[model.FeeHead], error)
	UpdateFeeHead(ctx context.Context, feeHead *model.FeeHead) error
	SoftDeleteFeeHead(ctx context.Context, tenantID, id uuid.UUID) error

	CreateFeeStructure(ctx context.Context, feeStructure *model.FeeStructure) error
	GetFeeStructure(ctx context.Context, tenantID, id uuid.UUID) (*model.FeeStructure, error)
	GetFeeStructureByCode(ctx context.Context, tenantID, academicYearID uuid.UUID, code string) (*model.FeeStructure, error)
	ListFeeStructures(ctx context.Context, tenantID uuid.UUID, filter model.FeeStructureFilter, params model.PaginationParams) (*model.PaginatedResult[model.FeeStructure], error)
	UpdateFeeStructure(ctx context.Context, feeStructure *model.FeeStructure) error
	SoftDeleteFeeStructure(ctx context.Context, tenantID, id uuid.UUID) error
	ReplaceFeeStructureItems(ctx context.Context, tenantID, feeStructureID uuid.UUID, items []model.FeeStructureItem) error
	ListFeeStructureItems(ctx context.Context, tenantID, feeStructureID uuid.UUID) ([]model.FeeStructureItem, error)

	CreateFeeAssignment(ctx context.Context, assignment *model.StudentFeeAssignment) error
	GetFeeAssignment(ctx context.Context, tenantID, id uuid.UUID) (*model.StudentFeeAssignment, error)
	ListFeeAssignments(ctx context.Context, tenantID uuid.UUID, filter model.FeeAssignmentFilter, params model.PaginationParams) (*model.PaginatedResult[model.StudentFeeAssignment], error)
	UpdateFeeAssignment(ctx context.Context, assignment *model.StudentFeeAssignment) error
	SoftDeleteFeeAssignment(ctx context.Context, tenantID, id uuid.UUID) error
	ListStudentsForAssignment(ctx context.Context, assignment *model.StudentFeeAssignment, onlyStudentIDs []uuid.UUID) ([]model.Student, error)

	ListActiveConcessions(ctx context.Context, tenantID, studentID, academicYearID uuid.UUID, asOf time.Time) ([]model.Concession, error)
	NextInvoiceSequence(ctx context.Context, tenantID, academicYearID uuid.UUID, prefix string) (int64, error)
	CreateInvoice(ctx context.Context, invoice *model.Invoice) error
	CreateInvoiceItems(ctx context.Context, items []model.InvoiceItem) error
	GetInvoice(ctx context.Context, tenantID, id uuid.UUID) (*model.Invoice, error)
	GetInvoiceByGenerationKey(ctx context.Context, tenantID uuid.UUID, generationKey string) (*model.Invoice, error)
	ListInvoices(ctx context.Context, tenantID uuid.UUID, filter model.InvoiceFilter, params model.PaginationParams) (*model.PaginatedResult[model.Invoice], error)
	ListInvoiceItems(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]model.InvoiceItem, error)
	ListStudentInvoices(ctx context.Context, tenantID, studentID uuid.UUID) ([]model.Invoice, error)
}

// PaymentRepository defines tenant-scoped payment, webhook, receipt, and ledger operations.
type PaymentRepository interface {
	CreatePaymentAttempt(ctx context.Context, attempt *model.PaymentAttempt) error
	GetPaymentAttempt(ctx context.Context, tenantID, id uuid.UUID) (*model.PaymentAttempt, error)
	GetPaymentAttemptByProviderOrderID(ctx context.Context, tenantID uuid.UUID, provider, providerOrderID string) (*model.PaymentAttempt, error)
	GetPaymentAttemptByProviderOrderIDAnyTenant(ctx context.Context, provider, providerOrderID string) (*model.PaymentAttempt, error)
	GetPaymentAttemptByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, idempotencyKey string) (*model.PaymentAttempt, error)
	UpdatePaymentAttemptProviderOrder(ctx context.Context, tenantID, id uuid.UUID, providerOrderID, checkoutURL, status string, metadata map[string]any) error
	UpdatePaymentAttemptStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error
	CreatePaymentAttemptAllocations(ctx context.Context, allocations []model.PaymentAllocation) error
	ListPaymentAttemptAllocations(ctx context.Context, tenantID, attemptID uuid.UUID) ([]model.PaymentAllocation, error)

	GetInvoiceForPayment(ctx context.Context, tenantID, invoiceID uuid.UUID) (*model.Invoice, error)
	ApplyInvoicePayment(ctx context.Context, tenantID, invoiceID uuid.UUID, amountPaise int64, asOf time.Time) (*model.Invoice, error)

	CreatePayment(ctx context.Context, payment *model.Payment) error
	GetPayment(ctx context.Context, tenantID, id uuid.UUID) (*model.Payment, error)
	GetPaymentByGatewayPaymentID(ctx context.Context, tenantID uuid.UUID, provider, gatewayPaymentID string) (*model.Payment, error)
	ListPayments(ctx context.Context, tenantID uuid.UUID, filter model.PaymentFilter, params model.PaginationParams) (*model.PaginatedResult[model.Payment], error)
	CreatePaymentAllocations(ctx context.Context, allocations []model.PaymentAllocation) error
	ListPaymentAllocations(ctx context.Context, tenantID, paymentID uuid.UUID) ([]model.PaymentAllocation, error)
	ListStudentPayments(ctx context.Context, tenantID, studentID uuid.UUID) ([]model.Payment, error)

	CreateGatewayWebhook(ctx context.Context, webhook *model.GatewayWebhook) error
	GetGatewayWebhookByEventID(ctx context.Context, provider, eventID string) (*model.GatewayWebhook, error)
	UpdateGatewayWebhookStatus(ctx context.Context, tenantID, id uuid.UUID, status, errorMessage string) error

	NextReceiptNumber(ctx context.Context, tenantID, academicYearID uuid.UUID, branchID *uuid.UUID, prefix string) (int64, error)
	CreateReceipt(ctx context.Context, receipt *model.Receipt) error
	GetReceipt(ctx context.Context, tenantID, id uuid.UUID) (*model.Receipt, error)
	GetReceiptByPaymentID(ctx context.Context, tenantID, paymentID uuid.UUID) (*model.Receipt, error)
	ListReceipts(ctx context.Context, tenantID uuid.UUID, filter model.ReceiptFilter, params model.PaginationParams) (*model.PaginatedResult[model.Receipt], error)
	ListStudentReceipts(ctx context.Context, tenantID, studentID uuid.UUID) ([]model.Receipt, error)

	CreateOfflinePaymentReference(ctx context.Context, ref *model.OfflinePaymentReference) error
	GetOfflinePaymentReferenceByPaymentID(ctx context.Context, tenantID, paymentID uuid.UUID) (*model.OfflinePaymentReference, error)

	CreatePaymentEvent(ctx context.Context, event *model.PaymentEvent) error
	ListPaymentEvents(ctx context.Context, tenantID uuid.UUID, filter model.PaymentEventFilter, params model.PaginationParams) (*model.PaginatedResult[model.PaymentEvent], error)
}

// OperationsRepository defines tenant-scoped reminders, notification logs, reports, jobs, and exports.
type OperationsRepository interface {
	CreateReminderTemplate(ctx context.Context, template *model.ReminderTemplate) error
	GetReminderTemplate(ctx context.Context, tenantID, id uuid.UUID) (*model.ReminderTemplate, error)
	GetReminderTemplateByCode(ctx context.Context, tenantID uuid.UUID, code string) (*model.ReminderTemplate, error)
	ListReminderTemplates(ctx context.Context, tenantID uuid.UUID, filter model.ReminderTemplateFilter, params model.PaginationParams) (*model.PaginatedResult[model.ReminderTemplate], error)
	UpdateReminderTemplate(ctx context.Context, template *model.ReminderTemplate) error
	SoftDeleteReminderTemplate(ctx context.Context, tenantID, id uuid.UUID) error

	CreateReminderRule(ctx context.Context, rule *model.ReminderRule) error
	GetReminderRule(ctx context.Context, tenantID, id uuid.UUID) (*model.ReminderRule, error)
	GetReminderRuleByCode(ctx context.Context, tenantID uuid.UUID, code string) (*model.ReminderRule, error)
	ListReminderRules(ctx context.Context, tenantID uuid.UUID, filter model.ReminderRuleFilter, params model.PaginationParams) (*model.PaginatedResult[model.ReminderRule], error)
	UpdateReminderRule(ctx context.Context, rule *model.ReminderRule) error
	SoftDeleteReminderRule(ctx context.Context, tenantID, id uuid.UUID) error

	ListReminderCandidates(ctx context.Context, tenantID uuid.UUID, filter model.ReminderCandidateFilter) ([]model.ReminderCandidate, error)
	CreateJob(ctx context.Context, job *model.Job) error
	GetJob(ctx context.Context, tenantID, id uuid.UUID) (*model.Job, error)
	ListTenantsWithDueJobs(ctx context.Context, jobType string, limit int) ([]uuid.UUID, error)
	ClaimDueJobs(ctx context.Context, tenantID uuid.UUID, jobType, workerID string, limit int) ([]model.Job, error)
	MarkJobSucceeded(ctx context.Context, tenantID, id uuid.UUID) error
	MarkJobFailed(ctx context.Context, tenantID, id uuid.UUID, lastError string, retryAt *time.Time) error

	CreateReminderLog(ctx context.Context, log *model.ReminderLog) error
	GetReminderLog(ctx context.Context, tenantID, id uuid.UUID) (*model.ReminderLog, error)
	ListReminderLogs(ctx context.Context, tenantID uuid.UUID, filter model.ReminderLogFilter, params model.PaginationParams) (*model.PaginatedResult[model.ReminderLog], error)
	UpdateReminderLogDelivery(ctx context.Context, log *model.ReminderLog) error
	CreateNotificationLog(ctx context.Context, log *model.NotificationLog) error

	GetDashboardSummary(ctx context.Context, tenantID uuid.UUID, asOf time.Time, recentLimit int) (*model.DashboardSummary, error)
	ListCollectionReport(ctx context.Context, tenantID uuid.UUID, filter model.ReportFilter, params model.PaginationParams) (*model.PaginatedResult[model.CollectionReportRow], error)
	ListDefaulterReport(ctx context.Context, tenantID uuid.UUID, filter model.ReportFilter, params model.PaginationParams) (*model.PaginatedResult[model.DefaulterReportRow], error)
	ListDueReport(ctx context.Context, tenantID uuid.UUID, filter model.ReportFilter, params model.PaginationParams) (*model.PaginatedResult[model.DueReportRow], error)
	ListFeeHeadCollectionReport(ctx context.Context, tenantID uuid.UUID, filter model.ReportFilter, params model.PaginationParams) (*model.PaginatedResult[model.FeeHeadCollectionRow], error)
	ListPaymentMethodReport(ctx context.Context, tenantID uuid.UUID, filter model.ReportFilter, params model.PaginationParams) (*model.PaginatedResult[model.PaymentMethodReportRow], error)

	CreateExportJob(ctx context.Context, job *model.ExportJob) error
	GetExportJob(ctx context.Context, tenantID, id uuid.UUID) (*model.ExportJob, error)
	ListExportJobs(ctx context.Context, tenantID uuid.UUID, filter model.ExportJobFilter, params model.PaginationParams) (*model.PaginatedResult[model.ExportJob], error)
	MarkExportJobProcessing(ctx context.Context, tenantID, id uuid.UUID) error
	CompleteExportJob(ctx context.Context, job *model.ExportJob) error
	FailExportJob(ctx context.Context, tenantID, id uuid.UUID, message string) error
}
