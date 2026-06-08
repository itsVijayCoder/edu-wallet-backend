-- Phase 5: reminders, notifications, reports, and exports.

CREATE TABLE IF NOT EXISTS reminder_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        VARCHAR(160) NOT NULL,
    code        VARCHAR(80) NOT NULL,
    channel     VARCHAR(30) NOT NULL DEFAULT 'email'
                CHECK (channel IN ('email', 'sms', 'whatsapp', 'in_app')),
    subject     VARCHAR(200) NOT NULL DEFAULT '',
    body        TEXT NOT NULL,
    tone        VARCHAR(30) NOT NULL DEFAULT 'polite'
                CHECK (tone IN ('polite', 'formal', 'urgent')),
    status      VARCHAR(20) NOT NULL DEFAULT 'active'
                CHECK (status IN ('active', 'inactive', 'archived')),
    metadata    JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    UNIQUE (tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_reminder_templates_active_code
    ON reminder_templates (tenant_id, lower(code))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_reminder_templates_tenant_channel_status
    ON reminder_templates (tenant_id, channel, status)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS reminder_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    template_id     UUID,
    name            VARCHAR(160) NOT NULL,
    code            VARCHAR(80) NOT NULL,
    channel         VARCHAR(30) NOT NULL DEFAULT 'email'
                    CHECK (channel IN ('email', 'sms', 'whatsapp', 'in_app')),
    trigger_type    VARCHAR(30) NOT NULL DEFAULT 'manual'
                    CHECK (trigger_type IN ('before_due', 'on_due', 'after_due', 'manual')),
    offset_days     INTEGER NOT NULL DEFAULT 0,
    target_statuses TEXT[] NOT NULL DEFAULT ARRAY['issued', 'partially_paid', 'overdue']::text[],
    status          VARCHAR(20) NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active', 'inactive', 'archived')),
    max_attempts    INTEGER NOT NULL DEFAULT 3,
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    CHECK (max_attempts > 0),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_reminder_rules_template
        FOREIGN KEY (tenant_id, template_id)
        REFERENCES reminder_templates(tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_reminder_rules_active_code
    ON reminder_rules (tenant_id, lower(code))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_reminder_rules_tenant_status_trigger
    ON reminder_rules (tenant_id, status, trigger_type)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    job_type        VARCHAR(60) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'queued'
                    CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'dead', 'cancelled')),
    priority        INTEGER NOT NULL DEFAULT 0,
    run_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    attempts        INTEGER NOT NULL DEFAULT 0,
    max_attempts    INTEGER NOT NULL DEFAULT 3,
    locked_at       TIMESTAMPTZ,
    locked_by       VARCHAR(120) NOT NULL DEFAULT '',
    idempotency_key VARCHAR(180),
    payload         JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_error      TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (attempts >= 0),
    CHECK (max_attempts > 0),
    UNIQUE (tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_jobs_tenant_idempotency
    ON jobs (tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_jobs_due_queue
    ON jobs (tenant_id, job_type, status, run_at, priority DESC, created_at)
    WHERE status IN ('queued', 'failed');

CREATE TABLE IF NOT EXISTS reminder_logs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    rule_id             UUID,
    template_id         UUID,
    job_id              UUID,
    invoice_id          UUID,
    student_id          UUID NOT NULL,
    guardian_id         UUID,
    channel             VARCHAR(30) NOT NULL
                        CHECK (channel IN ('email', 'sms', 'whatsapp', 'in_app')),
    recipient           VARCHAR(255) NOT NULL DEFAULT '',
    subject             VARCHAR(200) NOT NULL DEFAULT '',
    message             TEXT NOT NULL DEFAULT '',
    status              VARCHAR(20) NOT NULL DEFAULT 'queued'
                        CHECK (status IN ('queued', 'sent', 'failed', 'skipped')),
    provider            VARCHAR(60) NOT NULL DEFAULT '',
    provider_message_id VARCHAR(160) NOT NULL DEFAULT '',
    provider_response   JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message       TEXT NOT NULL DEFAULT '',
    scheduled_for       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    attempted_at        TIMESTAMPTZ,
    sent_at             TIMESTAMPTZ,
    attempt_count       INTEGER NOT NULL DEFAULT 0,
    created_by          UUID REFERENCES users(id),
    metadata            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (attempt_count >= 0),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_reminder_logs_rule
        FOREIGN KEY (tenant_id, rule_id)
        REFERENCES reminder_rules(tenant_id, id),
    CONSTRAINT fk_reminder_logs_template
        FOREIGN KEY (tenant_id, template_id)
        REFERENCES reminder_templates(tenant_id, id),
    CONSTRAINT fk_reminder_logs_job
        FOREIGN KEY (tenant_id, job_id)
        REFERENCES jobs(tenant_id, id),
    CONSTRAINT fk_reminder_logs_invoice
        FOREIGN KEY (tenant_id, invoice_id)
        REFERENCES invoices(tenant_id, id),
    CONSTRAINT fk_reminder_logs_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES students(tenant_id, id),
    CONSTRAINT fk_reminder_logs_guardian
        FOREIGN KEY (tenant_id, guardian_id)
        REFERENCES guardians(tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_reminder_logs_tenant_created
    ON reminder_logs (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_reminder_logs_invoice
    ON reminder_logs (tenant_id, invoice_id, created_at DESC)
    WHERE invoice_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_reminder_logs_student_status
    ON reminder_logs (tenant_id, student_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_reminder_logs_status_scheduled
    ON reminder_logs (tenant_id, status, scheduled_for);

CREATE TABLE IF NOT EXISTS notification_logs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    reminder_log_id     UUID,
    channel             VARCHAR(30) NOT NULL
                        CHECK (channel IN ('email', 'sms', 'whatsapp', 'in_app')),
    recipient           VARCHAR(255) NOT NULL,
    provider            VARCHAR(60) NOT NULL DEFAULT '',
    status              VARCHAR(20) NOT NULL
                        CHECK (status IN ('sent', 'failed', 'skipped')),
    provider_message_id VARCHAR(160) NOT NULL DEFAULT '',
    provider_response   JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message       TEXT NOT NULL DEFAULT '',
    attempted_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_notification_logs_reminder
        FOREIGN KEY (tenant_id, reminder_log_id)
        REFERENCES reminder_logs(tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_notification_logs_tenant_status
    ON notification_logs (tenant_id, status, attempted_at DESC);

CREATE INDEX IF NOT EXISTS idx_notification_logs_reminder
    ON notification_logs (tenant_id, reminder_log_id)
    WHERE reminder_log_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS export_jobs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    export_type  VARCHAR(40) NOT NULL
                 CHECK (export_type IN ('collections', 'defaulters', 'dues', 'payment_methods', 'fee_heads', 'offline_payments', 'receipt_register')),
    status       VARCHAR(20) NOT NULL DEFAULT 'queued'
                 CHECK (status IN ('queued', 'processing', 'succeeded', 'failed')),
    format       VARCHAR(20) NOT NULL DEFAULT 'csv'
                 CHECK (format IN ('csv')),
    params       JSONB NOT NULL DEFAULT '{}'::jsonb,
    file_name    VARCHAR(255) NOT NULL DEFAULT '',
    content_type VARCHAR(120) NOT NULL DEFAULT 'text/csv; charset=utf-8',
    content      BYTEA,
    row_count    INTEGER NOT NULL DEFAULT 0,
    requested_by UUID REFERENCES users(id),
    completed_at TIMESTAMPTZ,
    error_message TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (row_count >= 0),
    UNIQUE (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_export_jobs_tenant_status_created
    ON export_jobs (tenant_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_export_jobs_requested_by
    ON export_jobs (tenant_id, requested_by, created_at DESC)
    WHERE requested_by IS NOT NULL;

INSERT INTO permissions (code, name, category, description) VALUES
    ('reminders.manage', 'Manage Reminders', 'tenant', 'Create reminder templates, rules, and send reminders'),
    ('reports.view', 'View Reports', 'tenant', 'View dashboard and financial reports'),
    ('exports.manage', 'Manage Exports', 'tenant', 'Create and download report exports')
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
JOIN permissions p ON p.code IN ('reminders.manage', 'reports.view', 'exports.manage')
WHERE r.slug = 'admin'
ON CONFLICT DO NOTHING;
