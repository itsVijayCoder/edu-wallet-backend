-- Reverse: drop the guardian -> user link.

DROP INDEX IF EXISTS idx_guardians_active_user;
DROP INDEX IF EXISTS idx_guardians_tenant_user;

ALTER TABLE guardians
    DROP COLUMN IF EXISTS user_id;