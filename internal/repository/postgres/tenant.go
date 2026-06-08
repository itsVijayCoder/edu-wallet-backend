package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository"
)

var (
	_ repository.TenantRepository           = (*tenantRepository)(nil)
	_ repository.TenantMembershipRepository = (*tenantRepository)(nil)
)

type tenantRepository struct {
	db database.DBTX
}

func NewTenantRepository(db database.DBTX) repository.TenantRepository {
	return &tenantRepository{db: db}
}

func NewTenantMembershipRepository(db database.DBTX) repository.TenantMembershipRepository {
	return &tenantRepository{db: db}
}

var allowedTenantSortColumns = map[string]bool{
	"created_at": true,
	"name":       true,
	"slug":       true,
	"status":     true,
}

func (r *tenantRepository) Create(ctx context.Context, tenant *model.Tenant) error {
	const query = `INSERT INTO tenants (
			name, slug, legal_name, domain, contact_email, contact_phone, status,
			address_line1, address_line2, city, state, postal_code, country, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		tenant.Name,
		tenant.Slug,
		tenant.LegalName,
		tenant.Domain,
		tenant.ContactEmail,
		tenant.ContactPhone,
		tenant.Status,
		tenant.Address.Line1,
		tenant.Address.Line2,
		tenant.Address.City,
		tenant.Address.State,
		tenant.Address.PostalCode,
		tenant.Address.Country,
		mustJSON(tenant.Metadata),
	).Scan(&tenant.ID, &tenant.CreatedAt, &tenant.UpdatedAt)
}

func (r *tenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	const query = `SELECT id, name, slug, legal_name, domain, contact_email, contact_phone, status,
			address_line1, address_line2, city, state, postal_code, country, metadata,
			created_at, updated_at, deleted_at
		FROM tenants
		WHERE id = $1 AND deleted_at IS NULL`
	return r.scanTenant(ctx, query, id)
}

func (r *tenantRepository) GetBySlug(ctx context.Context, slug string) (*model.Tenant, error) {
	const query = `SELECT id, name, slug, legal_name, domain, contact_email, contact_phone, status,
			address_line1, address_line2, city, state, postal_code, country, metadata,
			created_at, updated_at, deleted_at
		FROM tenants
		WHERE lower(slug) = lower($1) AND deleted_at IS NULL`
	return r.scanTenant(ctx, query, slug)
}

func (r *tenantRepository) List(ctx context.Context, params model.PaginationParams) (*model.PaginatedResult[model.Tenant], error) {
	params.Normalize()

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM tenants WHERE deleted_at IS NULL`).Scan(&total); err != nil {
		return nil, err
	}

	sortCol := "created_at"
	if allowedTenantSortColumns[params.SortBy] {
		sortCol = params.SortBy
	}
	sortDir := "DESC"
	if strings.EqualFold(params.SortDir, "asc") {
		sortDir = "ASC"
	}

	query := fmt.Sprintf(`SELECT id, name, slug, legal_name, domain, contact_email, contact_phone, status,
			address_line1, address_line2, city, state, postal_code, country, metadata,
			created_at, updated_at, deleted_at
		FROM tenants
		WHERE deleted_at IS NULL
		ORDER BY %s %s
		LIMIT $1 OFFSET $2`, sortCol, sortDir)

	rows, err := r.db.Query(ctx, query, params.PageSize, params.Offset())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tenants := make([]model.Tenant, 0, params.PageSize)
	for rows.Next() {
		tenant, err := scanTenantRow(rows)
		if err != nil {
			return nil, err
		}
		tenants = append(tenants, *tenant)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return model.NewPaginatedResult(tenants, total, params.Page, params.PageSize), nil
}

func (r *tenantRepository) Update(ctx context.Context, tenant *model.Tenant) error {
	const query = `UPDATE tenants SET
			name = $1,
			slug = $2,
			legal_name = $3,
			domain = $4,
			contact_email = $5,
			contact_phone = $6,
			status = $7,
			address_line1 = $8,
			address_line2 = $9,
			city = $10,
			state = $11,
			postal_code = $12,
			country = $13,
			metadata = $14,
			updated_at = NOW()
		WHERE id = $15 AND deleted_at IS NULL
		RETURNING updated_at`

	return r.db.QueryRow(ctx, query,
		tenant.Name,
		tenant.Slug,
		tenant.LegalName,
		tenant.Domain,
		tenant.ContactEmail,
		tenant.ContactPhone,
		tenant.Status,
		tenant.Address.Line1,
		tenant.Address.Line2,
		tenant.Address.City,
		tenant.Address.State,
		tenant.Address.PostalCode,
		tenant.Address.Country,
		mustJSON(tenant.Metadata),
		tenant.ID,
	).Scan(&tenant.UpdatedAt)
}

func (r *tenantRepository) CreateBranch(ctx context.Context, branch *model.TenantBranch) error {
	const query = `INSERT INTO tenant_branches (
			tenant_id, name, code, contact_email, contact_phone, status,
			address_line1, address_line2, city, state, postal_code, country, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		branch.TenantID,
		branch.Name,
		branch.Code,
		branch.ContactEmail,
		branch.ContactPhone,
		branch.Status,
		branch.Address.Line1,
		branch.Address.Line2,
		branch.Address.City,
		branch.Address.State,
		branch.Address.PostalCode,
		branch.Address.Country,
		mustJSON(branch.Metadata),
	).Scan(&branch.ID, &branch.CreatedAt, &branch.UpdatedAt)
}

func (r *tenantRepository) ListBranches(ctx context.Context, tenantID uuid.UUID) ([]model.TenantBranch, error) {
	const query = `SELECT id, tenant_id, name, code, contact_email, contact_phone, status,
			address_line1, address_line2, city, state, postal_code, country, metadata,
			created_at, updated_at, deleted_at
		FROM tenant_branches
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY name ASC`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	branches := []model.TenantBranch{}
	for rows.Next() {
		branch, err := scanBranchRow(rows)
		if err != nil {
			return nil, err
		}
		branches = append(branches, *branch)
	}
	return branches, rows.Err()
}

func (r *tenantRepository) CreateMembership(ctx context.Context, membership *model.TenantMembership) error {
	const query = `INSERT INTO tenant_memberships (tenant_id, user_id, role_id, status, joined_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, user_id)
		DO UPDATE SET role_id = EXCLUDED.role_id, status = EXCLUDED.status, updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		membership.TenantID,
		membership.UserID,
		membership.RoleID,
		membership.Status,
		membership.JoinedAt,
	).Scan(&membership.ID, &membership.CreatedAt, &membership.UpdatedAt)
}

func (r *tenantRepository) GetByUserAndTenant(ctx context.Context, userID, tenantID uuid.UUID) (*model.TenantMembership, error) {
	const query = membershipQuery + `
		WHERE tm.user_id = $1
		  AND tm.tenant_id = $2
		  AND tm.status = 'active'
		  AND t.deleted_at IS NULL`

	membership, err := r.scanMembership(ctx, query, userID, tenantID)
	if err != nil || membership == nil {
		return membership, err
	}
	return r.withPermissions(ctx, membership)
}

func (r *tenantRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.TenantMembership, error) {
	const query = membershipQuery + `
		WHERE tm.user_id = $1
		  AND tm.status = 'active'
		  AND t.deleted_at IS NULL
		ORDER BY t.name ASC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}

	memberships := []model.TenantMembership{}
	for rows.Next() {
		membership, err := scanMembershipRow(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		memberships = append(memberships, *membership)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	for i := range memberships {
		if _, err := r.withPermissions(ctx, &memberships[i]); err != nil {
			return nil, err
		}
	}

	return memberships, nil
}

func (r *tenantRepository) ListPermissionsByRole(ctx context.Context, roleID uuid.UUID) ([]model.Permission, error) {
	const query = `SELECT p.id, p.code, p.name, p.category, p.description, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = $1
		ORDER BY p.code ASC`

	rows, err := r.db.Query(ctx, query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	permissions := []model.Permission{}
	for rows.Next() {
		var p model.Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.Category, &p.Description, &p.CreatedAt); err != nil {
			return nil, err
		}
		permissions = append(permissions, p)
	}
	return permissions, rows.Err()
}

const membershipQuery = `SELECT
		tm.id, tm.tenant_id, tm.user_id, tm.role_id, tm.status, tm.invited_at, tm.joined_at,
		tm.created_at, tm.updated_at,
		t.id, t.name, t.slug, t.legal_name, t.domain, t.contact_email, t.contact_phone, t.status,
		t.address_line1, t.address_line2, t.city, t.state, t.postal_code, t.country, t.metadata,
		t.created_at, t.updated_at, t.deleted_at,
		r.id, r.name, r.slug, r.description, r.scope, r.is_system, r.created_at, r.updated_at
	FROM tenant_memberships tm
	JOIN tenants t ON t.id = tm.tenant_id
	JOIN roles r ON r.id = tm.role_id`

func (r *tenantRepository) scanTenant(ctx context.Context, query string, args ...any) (*model.Tenant, error) {
	row := r.db.QueryRow(ctx, query, args...)
	tenant, err := scanTenantScanner(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return tenant, nil
}

func (r *tenantRepository) scanMembership(ctx context.Context, query string, args ...any) (*model.TenantMembership, error) {
	row := r.db.QueryRow(ctx, query, args...)
	membership, err := scanMembershipScanner(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return membership, nil
}

func (r *tenantRepository) withPermissions(ctx context.Context, membership *model.TenantMembership) (*model.TenantMembership, error) {
	if membership == nil {
		return nil, nil
	}
	permissions, err := r.ListPermissionsByRole(ctx, membership.RoleID)
	if err != nil {
		return nil, err
	}
	membership.Permissions = permissions
	return membership, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTenantScanner(row rowScanner) (*model.Tenant, error) {
	var tenant model.Tenant
	var metadata []byte
	err := row.Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Slug,
		&tenant.LegalName,
		&tenant.Domain,
		&tenant.ContactEmail,
		&tenant.ContactPhone,
		&tenant.Status,
		&tenant.Address.Line1,
		&tenant.Address.Line2,
		&tenant.Address.City,
		&tenant.Address.State,
		&tenant.Address.PostalCode,
		&tenant.Address.Country,
		&metadata,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&tenant.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	tenant.Metadata = parseJSON(metadata)
	return &tenant, nil
}

func scanTenantRow(rows pgx.Rows) (*model.Tenant, error) {
	return scanTenantScanner(rows)
}

func scanBranchRow(rows pgx.Rows) (*model.TenantBranch, error) {
	var branch model.TenantBranch
	var metadata []byte
	err := rows.Scan(
		&branch.ID,
		&branch.TenantID,
		&branch.Name,
		&branch.Code,
		&branch.ContactEmail,
		&branch.ContactPhone,
		&branch.Status,
		&branch.Address.Line1,
		&branch.Address.Line2,
		&branch.Address.City,
		&branch.Address.State,
		&branch.Address.PostalCode,
		&branch.Address.Country,
		&metadata,
		&branch.CreatedAt,
		&branch.UpdatedAt,
		&branch.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	branch.Metadata = parseJSON(metadata)
	return &branch, nil
}

func scanMembershipScanner(row rowScanner) (*model.TenantMembership, error) {
	var membership model.TenantMembership
	tenant := model.Tenant{}
	role := model.Role{}
	var metadata []byte
	err := row.Scan(
		&membership.ID,
		&membership.TenantID,
		&membership.UserID,
		&membership.RoleID,
		&membership.Status,
		&membership.InvitedAt,
		&membership.JoinedAt,
		&membership.CreatedAt,
		&membership.UpdatedAt,
		&tenant.ID,
		&tenant.Name,
		&tenant.Slug,
		&tenant.LegalName,
		&tenant.Domain,
		&tenant.ContactEmail,
		&tenant.ContactPhone,
		&tenant.Status,
		&tenant.Address.Line1,
		&tenant.Address.Line2,
		&tenant.Address.City,
		&tenant.Address.State,
		&tenant.Address.PostalCode,
		&tenant.Address.Country,
		&metadata,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&tenant.DeletedAt,
		&role.ID,
		&role.Name,
		&role.Slug,
		&role.Description,
		&role.Scope,
		&role.IsSystem,
		&role.CreatedAt,
		&role.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	tenant.Metadata = parseJSON(metadata)
	membership.Tenant = &tenant
	membership.Role = &role
	return &membership, nil
}

func scanMembershipRow(rows pgx.Rows) (*model.TenantMembership, error) {
	return scanMembershipScanner(rows)
}

func mustJSON(v map[string]any) []byte {
	if v == nil {
		return []byte(`{}`)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{}`)
	}
	return b
}

func parseJSON(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}
