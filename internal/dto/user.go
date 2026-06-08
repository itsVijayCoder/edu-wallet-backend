package dto

import "github.com/google/uuid"

type CreateUserRequest struct {
	Email     string   `json:"email"      binding:"required,email"`
	Password  string   `json:"password"   binding:"required,min=8"`
	FirstName string   `json:"first_name" binding:"required"`
	LastName  string   `json:"last_name"  binding:"required"`
	RoleSlugs []string `json:"roles"      binding:"required,min=1"`
}

type CreateTenantUserRequest struct {
	Email     string `json:"email"      binding:"required,email"`
	Password  string `json:"password"   binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name"  binding:"required"`
	RoleSlug  string `json:"role"       binding:"required,oneof=admin staff parents student"`
}

type TenantUserResponse struct {
	User       UserResponse `json:"user"`
	TenantID   uuid.UUID    `json:"tenant_id"`
	TenantRole string       `json:"tenant_role"`
}

type UpdateUserRequest struct {
	Email     *string  `json:"email"      binding:"omitempty,email"`
	FirstName *string  `json:"first_name"`
	LastName  *string  `json:"last_name"`
	Status    *string  `json:"status"     binding:"omitempty,oneof=active inactive"`
	RoleSlugs []string `json:"roles"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password"     binding:"required,min=8"`
}

type UpdateProfileRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
}

type SessionResponse struct {
	ID           string `json:"id"`
	IPAddress    string `json:"ip_address"`
	UserAgent    string `json:"user_agent"`
	LastActiveAt string `json:"last_active_at"`
	CreatedAt    string `json:"created_at"`
}
