package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository"
)

// Compile-time check: *sessionRepository implements repository.SessionRepository.
var _ repository.SessionRepository = (*sessionRepository)(nil)

type sessionRepository struct {
	db database.DBTX
}

// NewSessionRepository returns a SessionRepository backed by Postgres.
func NewSessionRepository(db database.DBTX) repository.SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	const query = `DELETE FROM refresh_tokens WHERE user_id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}
