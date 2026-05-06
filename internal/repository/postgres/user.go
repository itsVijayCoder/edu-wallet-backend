package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository"
)

// Compile-time check: *userRepository implements repository.UserRepository.
var _ repository.UserRepository = (*userRepository)(nil)

type userRepository struct {
	db database.DBTX
}

// NewUserRepository returns a UserRepository backed by Postgres.
func NewUserRepository(db database.DBTX) repository.UserRepository {
	return &userRepository{db: db}
}

// allowedSortColumns is the whitelist of columns that can appear in ORDER BY.
var allowedSortColumns = map[string]bool{
	"created_at": true,
	"email":      true,
	"first_name": true,
	"last_name":  true,
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	const query = `INSERT INTO users (email, password_hash, first_name, last_name, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		user.Email,
		user.PasswordHash,
		user.FirstName,
		user.LastName,
		user.Status,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	const query = `SELECT id, email, password_hash, first_name, last_name, status,
		last_login_at, created_at, updated_at, deleted_at
		FROM users WHERE id = $1 AND deleted_at IS NULL`

	user, err := r.scanUser(ctx, query, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	roles, err := r.getRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles
	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	const query = `SELECT id, email, password_hash, first_name, last_name, status,
		last_login_at, created_at, updated_at, deleted_at
		FROM users WHERE email = $1 AND deleted_at IS NULL`

	user, err := r.scanUser(ctx, query, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	roles, err := r.getRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles
	return user, nil
}

func (r *userRepository) List(ctx context.Context, params model.PaginationParams) (*model.PaginatedResult[model.User], error) {
	params.Normalize()

	// Count total rows.
	const countQuery = `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`
	var total int64
	if err := r.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, err
	}

	// Sanitize sort column.
	sortCol := "created_at"
	if allowedSortColumns[params.SortBy] {
		sortCol = params.SortBy
	}
	sortDir := "DESC"
	if strings.EqualFold(params.SortDir, "asc") {
		sortDir = "ASC"
	}

	listQuery := fmt.Sprintf(
		`SELECT id, email, password_hash, first_name, last_name, status,
			last_login_at, created_at, updated_at, deleted_at
		FROM users
		WHERE deleted_at IS NULL
		ORDER BY %s %s
		LIMIT $1 OFFSET $2`, sortCol, sortDir,
	)

	rows, err := r.db.Query(ctx, listQuery, params.PageSize, params.Offset())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(
			&u.ID, &u.Email, &u.PasswordHash,
			&u.FirstName, &u.LastName, &u.Status,
			&u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return model.NewPaginatedResult(users, total, params.Page, params.PageSize), nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	setClauses := []string{}
	args := []any{}
	argPos := 1

	if user.Email != "" {
		setClauses = append(setClauses, fmt.Sprintf("email = $%d", argPos))
		args = append(args, user.Email)
		argPos++
	}
	if user.PasswordHash != "" {
		setClauses = append(setClauses, fmt.Sprintf("password_hash = $%d", argPos))
		args = append(args, user.PasswordHash)
		argPos++
	}
	if user.FirstName != "" {
		setClauses = append(setClauses, fmt.Sprintf("first_name = $%d", argPos))
		args = append(args, user.FirstName)
		argPos++
	}
	if user.LastName != "" {
		setClauses = append(setClauses, fmt.Sprintf("last_name = $%d", argPos))
		args = append(args, user.LastName)
		argPos++
	}
	if user.Status != "" {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, user.Status)
		argPos++
	}

	if len(setClauses) == 0 {
		return nil // nothing to update
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	query := fmt.Sprintf(
		`UPDATE users SET %s WHERE id = $%d AND deleted_at IS NULL
		RETURNING updated_at`,
		strings.Join(setClauses, ", "), argPos,
	)
	args = append(args, user.ID)

	return r.db.QueryRow(ctx, query, args...).Scan(&user.UpdatedAt)
}

func (r *userRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	const query = `UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *userRepository) AssignRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	// Remove existing role assignments.
	const deleteQuery = `DELETE FROM user_roles WHERE user_id = $1`
	if _, err := r.db.Exec(ctx, deleteQuery, userID); err != nil {
		return err
	}

	// Insert new role assignments.
	const insertQuery = `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`
	for _, roleID := range roleIDs {
		if _, err := r.db.Exec(ctx, insertQuery, userID, roleID); err != nil {
			return err
		}
	}
	return nil
}

func (r *userRepository) GetRoles(ctx context.Context, userID uuid.UUID) ([]model.Role, error) {
	return r.getRoles(ctx, userID)
}

func (r *userRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	const query = `UPDATE users SET last_login_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// ---------- helpers ----------

// scanUser executes a single-row user query and returns nil, nil when not found.
func (r *userRepository) scanUser(ctx context.Context, query string, args ...any) (*model.User, error) {
	var u model.User
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&u.ID, &u.Email, &u.PasswordHash,
		&u.FirstName, &u.LastName, &u.Status,
		&u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// getRoles fetches the roles associated with a user.
func (r *userRepository) getRoles(ctx context.Context, userID uuid.UUID) ([]model.Role, error) {
	const query = `SELECT r.id, r.name, r.slug, r.description, r.created_at, r.updated_at
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []model.Role
	for rows.Next() {
		var role model.Role
		if err := rows.Scan(
			&role.ID, &role.Name, &role.Slug,
			&role.Description, &role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}
