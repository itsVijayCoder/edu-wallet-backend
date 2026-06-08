package model

import (
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID           uuid.UUID      `json:"id"`
	Name         string         `json:"name"`
	Slug         string         `json:"slug"`
	LegalName    string         `json:"legal_name"`
	Domain       *string        `json:"domain,omitempty"`
	ContactEmail *string        `json:"contact_email,omitempty"`
	ContactPhone *string        `json:"contact_phone,omitempty"`
	Status       string         `json:"status"`
	Address      Address        `json:"address"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    *time.Time     `json:"deleted_at,omitempty"`
}

type Address struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type TenantBranch struct {
	ID           uuid.UUID      `json:"id"`
	TenantID     uuid.UUID      `json:"tenant_id"`
	Name         string         `json:"name"`
	Code         string         `json:"code"`
	ContactEmail *string        `json:"contact_email,omitempty"`
	ContactPhone *string        `json:"contact_phone,omitempty"`
	Status       string         `json:"status"`
	Address      Address        `json:"address"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    *time.Time     `json:"deleted_at,omitempty"`
}

type Permission struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type TenantMembership struct {
	ID          uuid.UUID    `json:"id"`
	TenantID    uuid.UUID    `json:"tenant_id"`
	UserID      uuid.UUID    `json:"user_id"`
	RoleID      uuid.UUID    `json:"role_id"`
	Status      string       `json:"status"`
	InvitedAt   *time.Time   `json:"invited_at,omitempty"`
	JoinedAt    *time.Time   `json:"joined_at,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Tenant      *Tenant      `json:"tenant,omitempty"`
	Role        *Role        `json:"role,omitempty"`
	Permissions []Permission `json:"permissions,omitempty"`
}
