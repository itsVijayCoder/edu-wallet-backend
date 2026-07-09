-- Phase 7: link guardian contact records to parent login accounts.
--
-- A guardian record is a tenant-scoped contact for a student. Until now there
-- was no structural way to bind it to the users table, which meant the admin
-- could not tell whether a guardian had a login account, and a parent user
-- could not be reliably associated with their children.
--
-- The user_id is nullable + tenant-scoped to keep the guardian record usable
-- for contacts that exist purely for billing reminders without needing a
-- login. A partial unique index enforces a one-to-one link per tenant:
-- the same parent user cannot be bound to two guardian rows in the same
-- tenant, but a parent may be a guardian in multiple tenants.

ALTER TABLE guardians
    ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_guardians_tenant_user
    ON guardians (tenant_id, user_id)
    WHERE user_id IS NOT NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_guardians_active_user
    ON guardians (tenant_id, user_id)
    WHERE user_id IS NOT NULL AND deleted_at IS NULL;