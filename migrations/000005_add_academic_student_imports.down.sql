DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE code IN ('academic.manage', 'students.manage', 'guardians.manage', 'imports.manage')
);

DELETE FROM permissions
WHERE code IN ('academic.manage', 'students.manage', 'guardians.manage', 'imports.manage');

DROP TABLE IF EXISTS import_errors;
DROP TABLE IF EXISTS imports;
DROP TABLE IF EXISTS student_guardians;
DROP TABLE IF EXISTS guardians;
DROP TABLE IF EXISTS students;
DROP TABLE IF EXISTS sections;
DROP TABLE IF EXISTS classes;
DROP TABLE IF EXISTS academic_years;
