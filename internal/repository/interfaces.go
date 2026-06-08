package repository

import (
	"context"

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
