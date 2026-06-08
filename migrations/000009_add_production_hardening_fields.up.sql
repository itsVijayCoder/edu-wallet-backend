-- Phase 6: production hardening fields for provider retries, settlement, and reconciliation.

ALTER TABLE payment_attempts
    ADD COLUMN IF NOT EXISTS provider_retry_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS provider_last_error TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS reconciliation_status VARCHAR(30) NOT NULL DEFAULT 'not_required'
        CHECK (reconciliation_status IN ('not_required', 'pending', 'matched', 'mismatch', 'manually_reviewed')),
    ADD COLUMN IF NOT EXISTS reconciled_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS settlement_reference VARCHAR(160) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS settled_at TIMESTAMPTZ;

ALTER TABLE payment_attempts
    ADD CONSTRAINT payment_attempts_provider_retry_count_check
    CHECK (provider_retry_count >= 0) NOT VALID;

ALTER TABLE payment_attempts
    VALIDATE CONSTRAINT payment_attempts_provider_retry_count_check;

ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS reconciliation_status VARCHAR(30) NOT NULL DEFAULT 'pending'
        CHECK (reconciliation_status IN ('not_required', 'pending', 'matched', 'mismatch', 'manually_reviewed')),
    ADD COLUMN IF NOT EXISTS reconciled_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS settlement_reference VARCHAR(160) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS settled_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_payment_attempts_reconciliation
    ON payment_attempts (tenant_id, reconciliation_status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_payments_reconciliation
    ON payments (tenant_id, reconciliation_status, paid_at DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_payments_gateway_order
    ON payments (tenant_id, provider, gateway_order_id)
    WHERE gateway_order_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_payments_external_reference
    ON payments (tenant_id, external_reference)
    WHERE external_reference <> '';
