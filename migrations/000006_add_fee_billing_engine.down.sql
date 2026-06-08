DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE code IN ('fees.manage')
);

DELETE FROM permissions
WHERE code IN ('fees.manage');

DROP TABLE IF EXISTS invoice_items;
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS invoice_counters;
DROP TABLE IF EXISTS late_fee_rules;
DROP TABLE IF EXISTS concessions;
DROP TABLE IF EXISTS student_fee_assignments;
DROP TABLE IF EXISTS fee_structure_items;
DROP TABLE IF EXISTS fee_structures;
DROP TABLE IF EXISTS fee_heads;
