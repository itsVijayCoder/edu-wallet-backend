package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository"
)

// Compile-time check: *roleRepository implements repository.RoleRepository.
var _ repository.RoleRepository = (*roleRepository)(nil)

type roleRepository struct {
	db database.DBTX
}

// NewRoleRepository returns a RoleRepository backed by Postgres.
func NewRoleRepository(db database.DBTX) repository.RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Role, error) {
	const query = `SELECT id, name, slug, description, created_at, updated_at
		FROM roles WHERE id = $1`

	var role model.Role
	err := r.db.QueryRow(ctx, query, id).Scan(
		&role.ID,
		&role.Name,
		&role.Slug,
		&role.Description,
		&role.CreatedAt,
		&role.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) GetBySlug(ctx context.Context, slug string) (*model.Role, error) {
	const query = `SELECT id, name, slug, description, created_at, updated_at
		FROM roles WHERE slug = $1`

	var role model.Role
	err := r.db.QueryRow(ctx, query, slug).Scan(
		&role.ID,
		&role.Name,
		&role.Slug,
		&role.Description,
		&role.CreatedAt,
		&role.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) List(ctx context.Context) ([]model.Role, error) {
	const query = `SELECT id, name, slug, description, created_at, updated_at
		FROM roles ORDER BY name`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []model.Role
	for rows.Next() {
		var role model.Role
		if err := rows.Scan(
			&role.ID,
			&role.Name,
			&role.Slug,
			&role.Description,
			&role.CreatedAt,
			&role.UpdatedAt,
		); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}
