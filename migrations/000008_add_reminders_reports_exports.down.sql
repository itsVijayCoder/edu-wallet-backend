DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE code IN ('reminders.manage', 'reports.view', 'exports.manage')
);

DELETE FROM permissions
WHERE code IN ('reminders.manage', 'reports.view', 'exports.manage');

DROP TABLE IF EXISTS export_jobs;
DROP TABLE IF EXISTS notification_logs;
DROP TABLE IF EXISTS reminder_logs;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS reminder_rules;
DROP TABLE IF EXISTS reminder_templates;
