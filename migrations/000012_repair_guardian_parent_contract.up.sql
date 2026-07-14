-- Repair and complete the guardian/parent API schema.
--
-- The IF NOT EXISTS clauses intentionally make this migration safe for
-- databases where migration 000011 was recorded but its guardian user column
-- or indexes were removed manually. Earlier migrations remain the owners of
-- user_id and student_guardians; this migration only repairs missing objects
-- and adds the WhatsApp-specific preference.

ALTER TABLE guardians
    ADD COLUMN IF NOT EXISTS user_id UUID;

-- Match the intended ON DELETE SET NULL behavior before restoring the FK on a
-- drifted database that may contain references to users deleted out-of-band.
UPDATE guardians g
SET user_id = NULL
WHERE g.user_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM users u WHERE u.id = g.user_id);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'guardians'::regclass
          AND conname = 'guardians_user_id_fkey'
    ) THEN
        ALTER TABLE guardians
            ADD CONSTRAINT guardians_user_id_fkey
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_guardians_tenant_user
    ON guardians (tenant_id, user_id)
    WHERE user_id IS NOT NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_guardians_active_user
    ON guardians (tenant_id, user_id)
    WHERE user_id IS NOT NULL AND deleted_at IS NULL;

ALTER TABLE guardians
    ADD COLUMN IF NOT EXISTS opt_in_whatsapp BOOLEAN;

UPDATE guardians
SET opt_in_whatsapp = communication_opt_in
WHERE opt_in_whatsapp IS NULL;

ALTER TABLE guardians
    ALTER COLUMN opt_in_whatsapp SET DEFAULT TRUE,
    ALTER COLUMN opt_in_whatsapp SET NOT NULL;

CREATE TABLE IF NOT EXISTS student_guardians (
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    student_id   UUID NOT NULL,
    guardian_id  UUID NOT NULL,
    relationship VARCHAR(60) NOT NULL DEFAULT '',
    is_primary   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (student_id, guardian_id),
    CONSTRAINT fk_student_guardians_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_student_guardians_guardian
        FOREIGN KEY (tenant_id, guardian_id)
        REFERENCES guardians(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_student_guardians_guardian
    ON student_guardians (tenant_id, guardian_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_student_guardians_one_primary
    ON student_guardians (tenant_id, student_id)
    WHERE is_primary = TRUE;
