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
