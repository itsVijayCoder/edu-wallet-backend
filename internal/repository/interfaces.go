package repository

import (
	"context"

	"github.com/google/uuid"

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
