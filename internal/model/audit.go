package model

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID          uuid.UUID      `json:"id"`
	TenantID    *uuid.UUID     `json:"tenant_id,omitempty"`
	ActorUserID *uuid.UUID     `json:"actor_user_id,omitempty"`
	Action      string         `json:"action"`
	EntityType  string         `json:"entity_type"`
	EntityID    *uuid.UUID     `json:"entity_id,omitempty"`
	Summary     string         `json:"summary"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	RequestID   *string        `json:"request_id,omitempty"`
	IPAddress   *netip.Addr    `json:"ip_address,omitempty"`
	UserAgent   *string        `json:"user_agent,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}
