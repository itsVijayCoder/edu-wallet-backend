-- Phase 4: payments, webhooks, receipts, and ledger updates.

CREATE TABLE IF NOT EXISTS payment_attempts (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id),
    student_id         UUID NOT NULL,
    provider           VARCHAR(30) NOT NULL
                       CHECK (provider IN ('razorpay', 'fake', 'offline')),
    provider_order_id  VARCHAR(120),
    idempotency_key    VARCHAR(160),
    status             VARCHAR(30) NOT NULL DEFAULT 'created'
                       CHECK (status IN (
                           'created', 'pending', 'processing', 'success', 'failed', 'expired',
                           'cancelled', 'refunded', 'partially_refunded', 'manually_verified',
                           'reconciliation_mismatch', 'webhook_pending', 'settlement_pending', 'settled'
                       )),
    amount_paise       BIGINT NOT NULL,
    currency           CHAR(3) NOT NULL DEFAULT 'INR',
    checkout_url       TEXT NOT NULL DEFAULT '',
    expires_at         TIMESTAMPTZ,
    created_by         UUID REFERENCES users(id),
    metadata           JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (amount_paise > 0),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_payment_attempts_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES students(tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_attempts_provider_order
    ON payment_attempts (tenant_id, provider, provider_order_id)
    WHERE provider_order_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_attempts_idempotency
    ON payment_attempts (tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_payment_attempts_student_status
    ON payment_attempts (tenant_id, student_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS payment_attempt_allocations (
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    attempt_id    UUID NOT NULL,
    invoice_id    UUID NOT NULL,
    amount_paise  BIGINT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, attempt_id, invoice_id),
    CHECK (amount_paise > 0),
    CONSTRAINT fk_payment_attempt_allocations_attempt
        FOREIGN KEY (tenant_id, attempt_id)
        REFERENCES payment_attempts(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_payment_attempt_allocations_invoice
        FOREIGN KEY (tenant_id, invoice_id)
        REFERENCES invoices(tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_payment_attempt_allocations_invoice
    ON payment_attempt_allocations (tenant_id, invoice_id);

CREATE TABLE IF NOT EXISTS payments (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL REFERENCES tenants(id),
    attempt_id           UUID,
    student_id           UUID NOT NULL,
    provider             VARCHAR(30) NOT NULL
                         CHECK (provider IN ('razorpay', 'fake', 'offline')),
    payment_method       VARCHAR(30) NOT NULL DEFAULT 'online'
                         CHECK (payment_method IN (
                             'online', 'upi', 'card', 'netbanking', 'wallet',
                             'cash', 'cheque', 'dd', 'bank_transfer', 'other'
                         )),
    status               VARCHAR(30) NOT NULL
                         CHECK (status IN (
                             'created', 'pending', 'processing', 'success', 'failed', 'expired',
                             'cancelled', 'refunded', 'partially_refunded', 'manually_verified',
                             'reconciliation_mismatch', 'webhook_pending', 'settlement_pending', 'settled'
                         )),
    amount_paise         BIGINT NOT NULL,
    amount_applied_paise BIGINT NOT NULL DEFAULT 0,
    currency             CHAR(3) NOT NULL DEFAULT 'INR',
    gateway_order_id     VARCHAR(120),
    gateway_payment_id   VARCHAR(120),
    gateway_signature    TEXT NOT NULL DEFAULT '',
    external_reference   VARCHAR(160) NOT NULL DEFAULT '',
    paid_at              TIMESTAMPTZ,
    verified_at          TIMESTAMPTZ,
    received_by          UUID REFERENCES users(id),
    metadata             JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (amount_paise > 0),
    CHECK (amount_applied_paise >= 0 AND amount_applied_paise <= amount_paise),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_payments_attempt
        FOREIGN KEY (tenant_id, attempt_id)
        REFERENCES payment_attempts(tenant_id, id),
    CONSTRAINT fk_payments_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES students(tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_gateway_payment
    ON payments (tenant_id, provider, gateway_payment_id)
    WHERE gateway_payment_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_payments_student_status
    ON payments (tenant_id, student_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_payments_method_created
    ON payments (tenant_id, payment_method, created_at DESC);

CREATE TABLE IF NOT EXISTS payment_allocations (
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    payment_id    UUID NOT NULL,
    invoice_id    UUID NOT NULL,
    amount_paise  BIGINT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, payment_id, invoice_id),
    CHECK (amount_paise > 0),
    CONSTRAINT fk_payment_allocations_payment
        FOREIGN KEY (tenant_id, payment_id)
        REFERENCES payments(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_payment_allocations_invoice
        FOREIGN KEY (tenant_id, invoice_id)
        REFERENCES invoices(tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_payment_allocations_invoice
    ON payment_allocations (tenant_id, invoice_id);

CREATE TABLE IF NOT EXISTS gateway_webhooks (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id),
    provider          VARCHAR(30) NOT NULL
                      CHECK (provider IN ('razorpay', 'fake')),
    event_id          VARCHAR(160) NOT NULL,
    event_type        VARCHAR(120) NOT NULL,
    processing_status VARCHAR(30) NOT NULL DEFAULT 'received'
                      CHECK (processing_status IN ('received', 'processed', 'duplicate', 'failed', 'ignored')),
    payload           JSONB NOT NULL,
    signature         TEXT NOT NULL DEFAULT '',
    error_message     TEXT NOT NULL DEFAULT '',
    received_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at      TIMESTAMPTZ,
    UNIQUE (tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_gateway_webhooks_provider_event
    ON gateway_webhooks (provider, event_id);

CREATE INDEX IF NOT EXISTS idx_gateway_webhooks_tenant_status
    ON gateway_webhooks (tenant_id, processing_status, received_at DESC);

CREATE TABLE IF NOT EXISTS receipt_series (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id),
    academic_year_id UUID,
    branch_id        UUID REFERENCES tenant_branches(id),
    prefix           VARCHAR(40) NOT NULL,
    scope_key        VARCHAR(180) NOT NULL,
    next_number      BIGINT NOT NULL DEFAULT 1,
    status           VARCHAR(20) NOT NULL DEFAULT 'active'
                     CHECK (status IN ('active', 'inactive')),
    metadata         JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (next_number > 0),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_receipt_series_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_receipt_series_scope
    ON receipt_series (tenant_id, scope_key);

CREATE TABLE IF NOT EXISTS receipts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id),
    receipt_number   VARCHAR(80) NOT NULL,
    payment_id       UUID NOT NULL,
    student_id       UUID NOT NULL,
    academic_year_id UUID NOT NULL,
    branch_id        UUID REFERENCES tenant_branches(id),
    status           VARCHAR(20) NOT NULL DEFAULT 'issued'
                     CHECK (status IN ('issued', 'cancelled')),
    issue_date       DATE NOT NULL,
    currency         CHAR(3) NOT NULL DEFAULT 'INR',
    amount_paise     BIGINT NOT NULL,
    payment_method   VARCHAR(30) NOT NULL,
    issued_by        UUID REFERENCES users(id),
    issued_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cancelled_at     TIMESTAMPTZ,
    metadata         JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (amount_paise > 0),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_receipts_payment
        FOREIGN KEY (tenant_id, payment_id)
        REFERENCES payments(tenant_id, id),
    CONSTRAINT fk_receipts_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES students(tenant_id, id),
    CONSTRAINT fk_receipts_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_receipts_tenant_number
    ON receipts (tenant_id, receipt_number);

CREATE UNIQUE INDEX IF NOT EXISTS idx_receipts_payment
    ON receipts (tenant_id, payment_id);

CREATE INDEX IF NOT EXISTS idx_receipts_student_created
    ON receipts (tenant_id, student_id, created_at DESC);

CREATE TABLE IF NOT EXISTS offline_payment_references (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id),
    payment_id        UUID NOT NULL,
    payment_method    VARCHAR(30) NOT NULL
                      CHECK (payment_method IN ('cash', 'cheque', 'dd', 'bank_transfer', 'upi', 'other')),
    reference_number  VARCHAR(160) NOT NULL DEFAULT '',
    bank_name         VARCHAR(160) NOT NULL DEFAULT '',
    instrument_date   DATE,
    deposited_at      TIMESTAMPTZ,
    clearance_status  VARCHAR(20) NOT NULL DEFAULT 'cleared'
                      CHECK (clearance_status IN ('pending', 'cleared', 'bounced', 'cancelled')),
    remarks           TEXT NOT NULL DEFAULT '',
    metadata          JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_offline_payment_references_payment
        FOREIGN KEY (tenant_id, payment_id)
        REFERENCES payments(tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_offline_payment_references_payment
    ON offline_payment_references (tenant_id, payment_id);

CREATE INDEX IF NOT EXISTS idx_offline_payment_references_status
    ON offline_payment_references (tenant_id, clearance_status, created_at DESC);

CREATE TABLE IF NOT EXISTS payment_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    payment_id    UUID,
    attempt_id    UUID,
    receipt_id    UUID,
    student_id    UUID NOT NULL,
    event_type    VARCHAR(80) NOT NULL,
    status        VARCHAR(30) NOT NULL,
    amount_paise  BIGINT NOT NULL DEFAULT 0,
    message       TEXT NOT NULL DEFAULT '',
    metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (amount_paise >= 0),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_payment_events_payment
        FOREIGN KEY (tenant_id, payment_id)
        REFERENCES payments(tenant_id, id),
    CONSTRAINT fk_payment_events_attempt
        FOREIGN KEY (tenant_id, attempt_id)
        REFERENCES payment_attempts(tenant_id, id),
    CONSTRAINT fk_payment_events_receipt
        FOREIGN KEY (tenant_id, receipt_id)
        REFERENCES receipts(tenant_id, id),
    CONSTRAINT fk_payment_events_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES students(tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_payment_events_ticker
    ON payment_events (tenant_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_payment_events_student
    ON payment_events (tenant_id, student_id, occurred_at DESC);

INSERT INTO permissions (code, name, category, description) VALUES
    ('payments.manage', 'Manage Payments', 'tenant', 'Record payments, process webhooks, and manage receipts')
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
JOIN permissions p ON p.code = 'payments.manage'
WHERE r.slug = 'admin'
ON CONFLICT DO NOTHING;
