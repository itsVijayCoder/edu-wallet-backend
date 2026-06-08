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
