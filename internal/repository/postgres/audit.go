package postgres

import (
	"context"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository"
)

var _ repository.AuditRepository = (*auditRepository)(nil)

type auditRepository struct {
	db database.DBTX
}

func NewAuditRepository(db database.DBTX) repository.AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) Create(ctx context.Context, entry *model.AuditLog) error {
	const query = `INSERT INTO audit_logs (
			tenant_id, actor_user_id, action, entity_type, entity_id, summary,
			metadata, request_id, ip_address, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`

	var ipAddress *string
	if entry.IPAddress != nil {
		value := entry.IPAddress.String()
		ipAddress = &value
	}

	return r.db.QueryRow(ctx, query,
		entry.TenantID,
		entry.ActorUserID,
		entry.Action,
		entry.EntityType,
		entry.EntityID,
		entry.Summary,
		mustJSON(entry.Metadata),
		entry.RequestID,
		ipAddress,
		entry.UserAgent,
	).Scan(&entry.ID, &entry.CreatedAt)
}
