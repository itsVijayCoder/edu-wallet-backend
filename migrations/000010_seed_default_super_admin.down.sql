DELETE FROM user_roles
WHERE user_id IN (
    SELECT id
    FROM users
    WHERE email = 'admin@eduwallet.in'
)
AND role_id IN (
    SELECT id
    FROM roles
    WHERE slug = 'super_admin'
);

DELETE FROM users
WHERE email = 'admin@eduwallet.in'
  AND first_name = 'EduWallet'
  AND last_name = 'Owner';
