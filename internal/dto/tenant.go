package dto

import (
	"time"

	"github.com/google/uuid"
)

type AddressRequest struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type AddressResponse struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type CreateTenantRequest struct {
	Name         string               `json:"name" binding:"required,min=2,max=160"`
	Slug         string               `json:"slug" binding:"required,min=2,max=120"`
	LegalName    string               `json:"legal_name"`
	Domain       *string              `json:"domain" binding:"omitempty,hostname"`
	ContactEmail *string              `json:"contact_email" binding:"omitempty,email"`
	ContactPhone *string              `json:"contact_phone"`
	Status       string               `json:"status" binding:"omitempty,oneof=active inactive trial suspended"`
	Address      AddressRequest       `json:"address"`
	Metadata     map[string]any       `json:"metadata"`
	OwnerUserID  *uuid.UUID           `json:"owner_user_id"`
	Branch       *CreateBranchRequest `json:"branch"`
}

type UpdateTenantRequest struct {
	Name         *string         `json:"name" binding:"omitempty,min=2,max=160"`
	Slug         *string         `json:"slug" binding:"omitempty,min=2,max=120"`
	LegalName    *string         `json:"legal_name"`
	Domain       *string         `json:"domain" binding:"omitempty,hostname"`
	ContactEmail *string         `json:"contact_email" binding:"omitempty,email"`
	ContactPhone *string         `json:"contact_phone"`
	Status       *string         `json:"status" binding:"omitempty,oneof=active inactive trial suspended"`
	Address      *AddressRequest `json:"address"`
	Metadata     map[string]any  `json:"metadata"`
}

type CreateBranchRequest struct {
	Name         string         `json:"name" binding:"required,min=2,max=160"`
	Code         string         `json:"code" binding:"required,min=1,max=40"`
	ContactEmail *string        `json:"contact_email" binding:"omitempty,email"`
	ContactPhone *string        `json:"contact_phone"`
	Status       string         `json:"status" binding:"omitempty,oneof=active inactive"`
	Address      AddressRequest `json:"address"`
	Metadata     map[string]any `json:"metadata"`
}

type TenantResponse struct {
	ID           uuid.UUID        `json:"id"`
	Name         string           `json:"name"`
	Slug         string           `json:"slug"`
	LegalName    string           `json:"legal_name"`
	Domain       *string          `json:"domain,omitempty"`
	ContactEmail *string          `json:"contact_email,omitempty"`
	ContactPhone *string          `json:"contact_phone,omitempty"`
	Status       string           `json:"status"`
	Address      AddressResponse  `json:"address"`
	Metadata     map[string]any   `json:"metadata,omitempty"`
	Branches     []BranchResponse `json:"branches,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

type BranchResponse struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	Name         string          `json:"name"`
	Code         string          `json:"code"`
	ContactEmail *string         `json:"contact_email,omitempty"`
	ContactPhone *string         `json:"contact_phone,omitempty"`
	Status       string          `json:"status"`
	Address      AddressResponse `json:"address"`
	Metadata     map[string]any  `json:"metadata,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type TenantMembershipBrief struct {
	TenantID    uuid.UUID `json:"tenant_id"`
	TenantName  string    `json:"tenant_name"`
	TenantSlug  string    `json:"tenant_slug"`
	Role        string    `json:"role"`
	Permissions []string  `json:"permissions,omitempty"`
}
