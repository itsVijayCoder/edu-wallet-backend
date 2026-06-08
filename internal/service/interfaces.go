package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
)

type AuthService interface {
	Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error)
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, error)
	RefreshToken(ctx context.Context, req dto.RefreshRequest) (*dto.TokenPair, error)
	SelectTenant(ctx context.Context, userID uuid.UUID, req dto.SelectTenantRequest) (*dto.TokenPair, error)
	Logout(ctx context.Context, userID uuid.UUID) error
	ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error
}

type UserService interface {
	Create(ctx context.Context, req dto.CreateUserRequest) (*dto.UserResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*dto.UserResponse, error)
	List(ctx context.Context, params model.PaginationParams) (*model.PaginatedResult[dto.UserResponse], error)
	Update(ctx context.Context, id uuid.UUID, req dto.UpdateUserRequest) (*dto.UserResponse, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type SettingsService interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (*dto.UserResponse, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) (*dto.UserResponse, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) error
}

type EmailService interface {
	SendPasswordReset(ctx context.Context, to, token string) error
	SendWelcome(ctx context.Context, to, name string) error
}

type TenantService interface {
	Create(ctx context.Context, actorID uuid.UUID, req dto.CreateTenantRequest) (*dto.TenantResponse, error)
	List(ctx context.Context, params model.PaginationParams) (*model.PaginatedResult[dto.TenantResponse], error)
	GetByID(ctx context.Context, id uuid.UUID) (*dto.TenantResponse, error)
	Update(ctx context.Context, actorID, id uuid.UUID, req dto.UpdateTenantRequest) (*dto.TenantResponse, error)
	CreateBranch(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateBranchRequest) (*dto.BranchResponse, error)
	GetCurrent(ctx context.Context, tenantID uuid.UUID) (*dto.TenantResponse, error)
	UpdateCurrent(ctx context.Context, actorID, tenantID uuid.UUID, req dto.UpdateTenantRequest) (*dto.TenantResponse, error)
}

type AcademicService interface {
	CreateAcademicYear(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateAcademicYearRequest) (*dto.AcademicYearResponse, error)
	ListAcademicYears(ctx context.Context, tenantID uuid.UUID, filter model.AcademicYearFilter, params model.PaginationParams) (*model.PaginatedResult[dto.AcademicYearResponse], error)
	GetAcademicYear(ctx context.Context, tenantID, id uuid.UUID) (*dto.AcademicYearResponse, error)
	UpdateAcademicYear(ctx context.Context, actorID, tenantID, id uuid.UUID, req dto.UpdateAcademicYearRequest) (*dto.AcademicYearResponse, error)
	DeleteAcademicYear(ctx context.Context, actorID, tenantID, id uuid.UUID) error

	CreateClass(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateClassRequest) (*dto.ClassResponse, error)
	ListClasses(ctx context.Context, tenantID uuid.UUID, filter model.ClassFilter, params model.PaginationParams) (*model.PaginatedResult[dto.ClassResponse], error)
	GetClass(ctx context.Context, tenantID, id uuid.UUID) (*dto.ClassResponse, error)
	UpdateClass(ctx context.Context, actorID, tenantID, id uuid.UUID, req dto.UpdateClassRequest) (*dto.ClassResponse, error)
	DeleteClass(ctx context.Context, actorID, tenantID, id uuid.UUID) error

	CreateSection(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateSectionRequest) (*dto.SectionResponse, error)
	ListSections(ctx context.Context, tenantID uuid.UUID, filter model.SectionFilter, params model.PaginationParams) (*model.PaginatedResult[dto.SectionResponse], error)
	GetSection(ctx context.Context, tenantID, id uuid.UUID) (*dto.SectionResponse, error)
	UpdateSection(ctx context.Context, actorID, tenantID, id uuid.UUID, req dto.UpdateSectionRequest) (*dto.SectionResponse, error)
	DeleteSection(ctx context.Context, actorID, tenantID, id uuid.UUID) error

	CreateGuardian(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateGuardianRequest) (*dto.GuardianResponse, error)
	ListGuardians(ctx context.Context, tenantID uuid.UUID, filter model.GuardianFilter, params model.PaginationParams) (*model.PaginatedResult[dto.GuardianResponse], error)
	GetGuardian(ctx context.Context, tenantID, id uuid.UUID) (*dto.GuardianResponse, error)
	UpdateGuardian(ctx context.Context, actorID, tenantID, id uuid.UUID, req dto.UpdateGuardianRequest) (*dto.GuardianResponse, error)
	DeleteGuardian(ctx context.Context, actorID, tenantID, id uuid.UUID) error

	CreateStudent(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateStudentRequest) (*dto.StudentResponse, error)
	ListStudents(ctx context.Context, tenantID uuid.UUID, filter model.StudentFilter, params model.PaginationParams) (*model.PaginatedResult[dto.StudentResponse], error)
	GetStudent(ctx context.Context, tenantID, id uuid.UUID) (*dto.StudentResponse, error)
	UpdateStudent(ctx context.Context, actorID, tenantID, id uuid.UUID, req dto.UpdateStudentRequest) (*dto.StudentResponse, error)
	DeleteStudent(ctx context.Context, actorID, tenantID, id uuid.UUID) error
	LinkStudentGuardian(ctx context.Context, actorID, tenantID, studentID uuid.UUID, req dto.StudentGuardianRequest) (*dto.StudentResponse, error)
	UnlinkStudentGuardian(ctx context.Context, actorID, tenantID, studentID, guardianID uuid.UUID) (*dto.StudentResponse, error)

	StudentImportTemplate() string
	PreviewStudentImport(ctx context.Context, actorID, tenantID uuid.UUID, filename string, csvData []byte) (*dto.StudentImportPreviewResponse, error)
	CommitStudentImport(ctx context.Context, actorID, tenantID uuid.UUID, req dto.StudentImportCommitRequest) (*dto.StudentImportCommitResponse, error)
	ListImports(ctx context.Context, tenantID uuid.UUID, filter model.ImportFilter, params model.PaginationParams) (*model.PaginatedResult[dto.ImportResponse], error)
}

type BillingService interface {
	CreateFeeHead(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateFeeHeadRequest) (*dto.FeeHeadResponse, error)
	ListFeeHeads(ctx context.Context, tenantID uuid.UUID, filter model.FeeHeadFilter, params model.PaginationParams) (*model.PaginatedResult[dto.FeeHeadResponse], error)
	GetFeeHead(ctx context.Context, tenantID, id uuid.UUID) (*dto.FeeHeadResponse, error)
	UpdateFeeHead(ctx context.Context, actorID, tenantID, id uuid.UUID, req dto.UpdateFeeHeadRequest) (*dto.FeeHeadResponse, error)
	DeleteFeeHead(ctx context.Context, actorID, tenantID, id uuid.UUID) error

	CreateFeeStructure(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateFeeStructureRequest) (*dto.FeeStructureResponse, error)
	ListFeeStructures(ctx context.Context, tenantID uuid.UUID, filter model.FeeStructureFilter, params model.PaginationParams) (*model.PaginatedResult[dto.FeeStructureResponse], error)
	GetFeeStructure(ctx context.Context, tenantID, id uuid.UUID) (*dto.FeeStructureResponse, error)
	UpdateFeeStructure(ctx context.Context, actorID, tenantID, id uuid.UUID, req dto.UpdateFeeStructureRequest) (*dto.FeeStructureResponse, error)
	DeleteFeeStructure(ctx context.Context, actorID, tenantID, id uuid.UUID) error

	CreateFeeAssignment(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateFeeAssignmentRequest) (*dto.FeeAssignmentResponse, error)
	GenerateInvoices(ctx context.Context, actorID, tenantID uuid.UUID, req dto.GenerateInvoicesRequest) (*dto.GenerateInvoicesResponse, error)
	ListInvoices(ctx context.Context, tenantID uuid.UUID, filter model.InvoiceFilter, params model.PaginationParams) (*model.PaginatedResult[dto.InvoiceResponse], error)
	GetInvoice(ctx context.Context, tenantID, id uuid.UUID) (*dto.InvoiceResponse, error)
	GetStudentLedger(ctx context.Context, tenantID, studentID uuid.UUID) (*dto.StudentLedgerResponse, error)
	GetParentChildDues(ctx context.Context, tenantID, studentID uuid.UUID) (*dto.ParentDuesResponse, error)
}
