DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE code IN ('payments.manage')
);

DELETE FROM permissions
WHERE code IN ('payments.manage');

DROP TABLE IF EXISTS payment_events;
DROP TABLE IF EXISTS offline_payment_references;
DROP TABLE IF EXISTS receipts;
DROP TABLE IF EXISTS receipt_series;
DROP TABLE IF EXISTS gateway_webhooks;
DROP TABLE IF EXISTS payment_allocations;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS payment_attempt_allocations;
DROP TABLE IF EXISTS payment_attempts;
