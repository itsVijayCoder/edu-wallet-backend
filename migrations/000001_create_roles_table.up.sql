-- Migration: Create roles table
-- Convention: UUID primary keys with gen_random_uuid()
-- Convention: All tables have created_at and updated_at timestamps

CREATE TABLE IF NOT EXISTS roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Seed default roles
-- Modify these to match your application's role model
INSERT INTO roles (name, slug, description) VALUES
    ('Super Admin', 'super_admin', 'Super Admin role'),
    ('Admin', 'admin', 'Admin role'),
    ('Parents', 'parents', 'Parents role'),
    ('Student', 'student', 'Student role'),
    ('Staff', 'staff', 'Staff role')
;
