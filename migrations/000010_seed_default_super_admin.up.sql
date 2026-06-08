-- Bootstrap the product owner/developer account.
-- This account owns platform-level onboarding and can create tenants/admins.

WITH upsert_user AS (
    INSERT INTO users (email, password_hash, first_name, last_name, status)
    VALUES (
        'admin@eduwallet.in',
        '$2a$12$.J1ffjpiZ4cWpOMO3AO6H.6HUVwaxyIx.tlBg0CCtqwBo7wwNFSi2',
        'EduWallet',
        'Owner',
        'active'
    )
    ON CONFLICT (email) DO UPDATE
    SET password_hash = EXCLUDED.password_hash,
        first_name = EXCLUDED.first_name,
        last_name = EXCLUDED.last_name,
        status = 'active',
        deleted_at = NULL,
        updated_at = NOW()
    RETURNING id
),
super_admin_role AS (
    SELECT id FROM roles WHERE slug = 'super_admin'
)
INSERT INTO user_roles (user_id, role_id)
SELECT upsert_user.id, super_admin_role.id
FROM upsert_user
CROSS JOIN super_admin_role
ON CONFLICT DO NOTHING;
