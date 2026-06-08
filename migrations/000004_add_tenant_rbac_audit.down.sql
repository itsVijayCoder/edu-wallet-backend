DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS tenant_memberships;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS tenant_branches;
DROP TABLE IF EXISTS tenants;

ALTER TABLE roles
    DROP COLUMN IF EXISTS is_system,
    DROP COLUMN IF EXISTS scope;
