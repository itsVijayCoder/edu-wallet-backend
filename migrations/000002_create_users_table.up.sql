-- Migration: Create users table
-- Convention: UUID primary keys (not auto-increment) for distributed safety
-- Convention: Email is UNIQUE and indexed for login lookups
-- Convention: Soft delete via deleted_at (NULL = active, timestamp = deleted)
-- Convention: Status enum via CHECK constraint
-- Convention: user_roles junction table for many-to-many

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    first_name    VARCHAR(100) NOT NULL DEFAULT '',
    last_name     VARCHAR(100) NOT NULL DEFAULT '',
    status        VARCHAR(20)  NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active', 'inactive', 'invited')),
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ          -- NULL = not deleted (soft delete convention)
);

-- Index on email for fast login lookups
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email) WHERE deleted_at IS NULL;

-- Junction table for user-role many-to-many relationship
-- Convention: Composite primary key, CASCADE deletes
CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);
