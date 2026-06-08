-- Phase 2 core: academic setup, students, guardians, and student imports.

CREATE TABLE IF NOT EXISTS academic_years (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        VARCHAR(120) NOT NULL,
    code        VARCHAR(40) NOT NULL,
    start_date  DATE NOT NULL,
    end_date    DATE NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'active'
                CHECK (status IN ('active', 'inactive', 'closed')),
    is_active   BOOLEAN NOT NULL DEFAULT FALSE,
    metadata    JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    CHECK (end_date >= start_date),
    UNIQUE (tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_academic_years_active_code
    ON academic_years (tenant_id, lower(code))
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_academic_years_one_current
    ON academic_years (tenant_id)
    WHERE is_active = TRUE AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_academic_years_tenant_status
    ON academic_years (tenant_id, status)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS classes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        VARCHAR(120) NOT NULL,
    code        VARCHAR(40) NOT NULL,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    status      VARCHAR(20) NOT NULL DEFAULT 'active'
                CHECK (status IN ('active', 'inactive')),
    metadata    JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    UNIQUE (tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_classes_active_code
    ON classes (tenant_id, lower(code))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_classes_tenant_status_sort
    ON classes (tenant_id, status, sort_order)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS sections (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id),
    academic_year_id UUID NOT NULL,
    class_id         UUID NOT NULL,
    branch_id        UUID REFERENCES tenant_branches(id),
    name             VARCHAR(120) NOT NULL,
    code             VARCHAR(40) NOT NULL,
    capacity         INTEGER,
    status           VARCHAR(20) NOT NULL DEFAULT 'active'
                     CHECK (status IN ('active', 'inactive')),
    metadata         JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ,
    CHECK (capacity IS NULL OR capacity >= 0),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_sections_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id),
    CONSTRAINT fk_sections_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES classes(tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sections_active_code
    ON sections (tenant_id, academic_year_id, class_id, lower(code))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_sections_class_year
    ON sections (tenant_id, class_id, academic_year_id, status)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS students (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID NOT NULL REFERENCES tenants(id),
    academic_year_id      UUID NOT NULL,
    class_id              UUID NOT NULL,
    section_id            UUID NOT NULL,
    branch_id             UUID REFERENCES tenant_branches(id),
    admission_number      VARCHAR(80) NOT NULL,
    first_name            VARCHAR(100) NOT NULL,
    last_name             VARCHAR(100) NOT NULL DEFAULT '',
    roll_number           VARCHAR(40),
    status                VARCHAR(20) NOT NULL DEFAULT 'active'
                          CHECK (status IN ('active', 'inactive', 'transferred', 'graduated')),
    category              VARCHAR(30) NOT NULL DEFAULT 'general'
                          CHECK (category IN ('general', 'scholarship', 'staff_child', 'sibling', 'custom')),
    phone                 VARCHAR(30),
    email                 VARCHAR(255),
    address_line1         VARCHAR(255) NOT NULL DEFAULT '',
    address_line2         VARCHAR(255) NOT NULL DEFAULT '',
    city                  VARCHAR(120) NOT NULL DEFAULT '',
    state                 VARCHAR(120) NOT NULL DEFAULT '',
    postal_code           VARCHAR(20) NOT NULL DEFAULT '',
    country               VARCHAR(120) NOT NULL DEFAULT 'India',
    opening_balance_paise BIGINT NOT NULL DEFAULT 0,
    currency              CHAR(3) NOT NULL DEFAULT 'INR',
    metadata              JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at            TIMESTAMPTZ,
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_students_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id),
    CONSTRAINT fk_students_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES classes(tenant_id, id),
    CONSTRAINT fk_students_section
        FOREIGN KEY (tenant_id, section_id)
        REFERENCES sections(tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_students_active_admission_number
    ON students (tenant_id, lower(admission_number))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_students_tenant_class_section
    ON students (tenant_id, academic_year_id, class_id, section_id, status)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_students_tenant_status
    ON students (tenant_id, status)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_students_search_name
    ON students (tenant_id, lower(first_name), lower(last_name))
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS guardians (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL REFERENCES tenants(id),
    name                     VARCHAR(160) NOT NULL,
    relationship             VARCHAR(60) NOT NULL DEFAULT '',
    phone                    VARCHAR(30),
    whatsapp_phone           VARCHAR(30),
    email                    VARCHAR(255),
    preferred_language       VARCHAR(40) NOT NULL DEFAULT 'en',
    communication_opt_in     BOOLEAN NOT NULL DEFAULT TRUE,
    address_line1            VARCHAR(255) NOT NULL DEFAULT '',
    address_line2            VARCHAR(255) NOT NULL DEFAULT '',
    city                     VARCHAR(120) NOT NULL DEFAULT '',
    state                    VARCHAR(120) NOT NULL DEFAULT '',
    postal_code              VARCHAR(20) NOT NULL DEFAULT '',
    country                  VARCHAR(120) NOT NULL DEFAULT 'India',
    metadata                 JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at               TIMESTAMPTZ,
    UNIQUE (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_guardians_tenant_phone
    ON guardians (tenant_id, phone)
    WHERE phone IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_guardians_tenant_email
    ON guardians (tenant_id, lower(email))
    WHERE email IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_guardians_tenant_name
    ON guardians (tenant_id, lower(name))
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS student_guardians (
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    student_id   UUID NOT NULL,
    guardian_id  UUID NOT NULL,
    relationship VARCHAR(60) NOT NULL DEFAULT '',
    is_primary   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (student_id, guardian_id),
    CONSTRAINT fk_student_guardians_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_student_guardians_guardian
        FOREIGN KEY (tenant_id, guardian_id)
        REFERENCES guardians(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_student_guardians_guardian
    ON student_guardians (tenant_id, guardian_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_student_guardians_one_primary
    ON student_guardians (tenant_id, student_id)
    WHERE is_primary = TRUE;

CREATE TABLE IF NOT EXISTS imports (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id),
    import_type      VARCHAR(40) NOT NULL,
    status           VARCHAR(20) NOT NULL DEFAULT 'previewed'
                     CHECK (status IN ('previewed', 'committed', 'failed')),
    source_filename  VARCHAR(255) NOT NULL DEFAULT '',
    total_rows       INTEGER NOT NULL DEFAULT 0,
    valid_rows       INTEGER NOT NULL DEFAULT 0,
    invalid_rows     INTEGER NOT NULL DEFAULT 0,
    committed_rows   INTEGER NOT NULL DEFAULT 0,
    created_by       UUID REFERENCES users(id),
    metadata         JSONB NOT NULL DEFAULT '{}'::jsonb,
    committed_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_imports_tenant_created
    ON imports (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_imports_tenant_status
    ON imports (tenant_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS import_errors (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    import_id   UUID NOT NULL REFERENCES imports(id) ON DELETE CASCADE,
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    row_number   INTEGER NOT NULL,
    field        VARCHAR(120) NOT NULL DEFAULT '',
    message      TEXT NOT NULL,
    raw_data     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_import_errors_import_row
    ON import_errors (import_id, row_number);

CREATE INDEX IF NOT EXISTS idx_import_errors_tenant_created
    ON import_errors (tenant_id, created_at DESC);

INSERT INTO permissions (code, name, category, description) VALUES
    ('academic.manage', 'Manage Academic Setup', 'tenant', 'Create and update academic years, classes, and sections'),
    ('students.manage', 'Manage Students', 'tenant', 'Create, update, and list tenant students'),
    ('guardians.manage', 'Manage Guardians', 'tenant', 'Create, update, and list tenant guardians'),
    ('imports.manage', 'Manage Imports', 'tenant', 'Preview and commit student imports')
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    category = EXCLUDED.category,
    description = EXCLUDED.description;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.slug = 'super_admin'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('academic.manage', 'students.manage', 'guardians.manage', 'imports.manage')
WHERE r.slug = 'admin'
ON CONFLICT DO NOTHING;
