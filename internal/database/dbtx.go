package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX is satisfied by both *pgxpool.Pool and pgx.Tx, allowing repositories
// to work identically inside or outside a database transaction.
//
// Usage without transaction:
//
//	repo := postgres.NewUserRepository(pool)
//	repo.Create(ctx, user)
//
// Usage with transaction:
//
//	database.WithTx(ctx, pool, func(tx pgx.Tx) error {
//	    userRepo := postgres.NewUserRepository(tx)
//	    roleRepo := postgres.NewRoleRepository(tx)
//	    userRepo.Create(ctx, user)
//	    roleRepo.Assign(ctx, user.ID, roleID)
//	    return nil // commit
//	})
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// WithTx executes fn inside a database transaction.
// It automatically rolls back on error and commits on success.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
