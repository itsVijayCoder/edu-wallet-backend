DROP INDEX IF EXISTS idx_payments_external_reference;
DROP INDEX IF EXISTS idx_payments_gateway_order;
DROP INDEX IF EXISTS idx_payments_reconciliation;
DROP INDEX IF EXISTS idx_payment_attempts_reconciliation;

ALTER TABLE payments
    DROP COLUMN IF EXISTS settled_at,
    DROP COLUMN IF EXISTS settlement_reference,
    DROP COLUMN IF EXISTS reconciled_at,
    DROP COLUMN IF EXISTS reconciliation_status;

ALTER TABLE payment_attempts
    DROP CONSTRAINT IF EXISTS payment_attempts_provider_retry_count_check;

ALTER TABLE payment_attempts
    DROP COLUMN IF EXISTS settled_at,
    DROP COLUMN IF EXISTS settlement_reference,
    DROP COLUMN IF EXISTS reconciled_at,
    DROP COLUMN IF EXISTS reconciliation_status,
    DROP COLUMN IF EXISTS provider_last_error,
    DROP COLUMN IF EXISTS provider_retry_count;
