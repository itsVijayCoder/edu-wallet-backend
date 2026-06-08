-- Phase 1 foundation: tenants, tenant-scoped RBAC, and immutable audit logs.

ALTER TABLE roles
    ADD COLUMN scope VARCHAR(20) NOT NULL DEFAULT 'tenant'
        CHECK (scope IN ('platform', 'tenant')),
    ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT TRUE;

UPDATE roles
SET scope = 'platform'
WHERE slug = 'super_admin';

UPDATE roles
SET scope = 'tenant'
WHERE slug <> 'super_admin';

CREATE TABLE IF NOT EXISTS tenants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(160) NOT NULL,
    slug          VARCHAR(120) NOT NULL,
    legal_name    VARCHAR(200) NOT NULL DEFAULT '',
    domain        VARCHAR(255),
    contact_email VARCHAR(255),
    contact_phone VARCHAR(30),
    status        VARCHAR(20) NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active', 'inactive', 'trial', 'suspended')),
    address_line1 VARCHAR(255) NOT NULL DEFAULT '',
    address_line2 VARCHAR(255) NOT NULL DEFAULT '',
    city          VARCHAR(120) NOT NULL DEFAULT '',
    state         VARCHAR(120) NOT NULL DEFAULT '',
    postal_code   VARCHAR(20) NOT NULL DEFAULT '',
    country       VARCHAR(120) NOT NULL DEFAULT 'India',
    metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_active_slug
    ON tenants (lower(slug))
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_active_domain
    ON tenants (lower(domain))
    WHERE domain IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tenants_status
    ON tenants (status)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS tenant_branches (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    name          VARCHAR(160) NOT NULL,
    code          VARCHAR(40) NOT NULL,
    contact_email VARCHAR(255),
    contact_phone VARCHAR(30),
    status        VARCHAR(20) NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active', 'inactive')),
    address_line1 VARCHAR(255) NOT NULL DEFAULT '',
    address_line2 VARCHAR(255) NOT NULL DEFAULT '',
    city          VARCHAR(120) NOT NULL DEFAULT '',
    state         VARCHAR(120) NOT NULL DEFAULT '',
    postal_code   VARCHAR(20) NOT NULL DEFAULT '',
    country       VARCHAR(120) NOT NULL DEFAULT 'India',
    metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenant_branches_active_code
    ON tenant_branches (tenant_id, lower(code))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tenant_branches_tenant
    ON tenant_branches (tenant_id)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(120) NOT NULL UNIQUE,
    name        VARCHAR(160) NOT NULL,
    category    VARCHAR(80) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_permission
    ON role_permissions (permission_id);

CREATE TABLE IF NOT EXISTS tenant_memberships (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    user_id     UUID NOT NULL REFERENCES users(id),
    role_id     UUID NOT NULL REFERENCES roles(id),
    status      VARCHAR(20) NOT NULL DEFAULT 'active'
                CHECK (status IN ('active', 'inactive', 'invited')),
    invited_at  TIMESTAMPTZ,
    joined_at   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_tenant_memberships_tenant_user
    ON tenant_memberships (tenant_id, user_id);

CREATE INDEX IF NOT EXISTS idx_tenant_memberships_user_status
    ON tenant_memberships (user_id, status);

CREATE INDEX IF NOT EXISTS idx_tenant_memberships_tenant_role
    ON tenant_memberships (tenant_id, role_id);

CREATE TABLE IF NOT EXISTS audit_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID REFERENCES tenants(id),
    actor_user_id UUID REFERENCES users(id),
    action        VARCHAR(120) NOT NULL,
    entity_type   VARCHAR(120) NOT NULL,
    entity_id     UUID,
    summary       TEXT NOT NULL DEFAULT '',
    metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
    request_id    VARCHAR(120),
    ip_address    INET,
    user_agent    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_created
    ON audit_logs (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_created
    ON audit_logs (actor_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_entity
    ON audit_logs (entity_type, entity_id);

INSERT INTO permissions (code, name, category, description) VALUES
    ('platform.tenants.manage', 'Manage Tenants', 'platform', 'Create and update tenant accounts'),
    ('tenant.read', 'Read Tenant', 'tenant', 'Read current tenant profile'),
    ('tenant.update', 'Update Tenant', 'tenant', 'Update current tenant profile'),
    ('branches.create', 'Create Branches', 'tenant', 'Create tenant branches'),
    ('users.manage', 'Manage Users', 'tenant', 'Create and manage tenant users')
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    category = EXCLUDED.category,
    description = EXCLUDED.description;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.slug = 'super_admin'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('tenant.read', 'tenant.update', 'branches.create', 'users.manage')
WHERE r.slug = 'admin'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'tenant.read'
WHERE r.slug IN ('staff', 'parents')
ON CONFLICT DO NOTHING;
