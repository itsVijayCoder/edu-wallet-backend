package dto

import (
	"time"

	"github.com/google/uuid"
)

type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type SendOTPRequest struct {
	Phone      string `json:"phone" binding:"required,e164"`
	TenantSlug string `json:"tenant_slug" binding:"omitempty,min=2,max=100"`
}

type VerifyOTPRequest struct {
	Phone string `json:"phone" binding:"required,e164"`
	OTP   string `json:"otp" binding:"required,numeric,min=4,max=6"`
}

type SendOTPResponse struct {
	Message          string `json:"message"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type RegisterRequest struct {
	Email     string `json:"email"      binding:"required,email"`
	Password  string `json:"password"   binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name"  binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type SelectTenantRequest struct {
	TenantID uuid.UUID `json:"tenant_id" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"        binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type TokenPair struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

type LoginResponse struct {
	TokenPair
	User    UserResponse            `json:"user"`
	Tenants []TenantMembershipBrief `json:"tenants,omitempty"`
}

type UserResponse struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	Name        string    `json:"name"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
}
