-- Phase 3 core: fee setup, assignments, invoice generation, and ledger reads.

CREATE TABLE IF NOT EXISTS fee_heads (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id),
    name             VARCHAR(160) NOT NULL,
    code             VARCHAR(60) NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    category         VARCHAR(40) NOT NULL DEFAULT 'tuition'
                     CHECK (category IN (
                         'admission', 'tuition', 'term', 'exam', 'transport', 'hostel',
                         'lab', 'library', 'uniform_books', 'fine', 'activity', 'sports',
                         'development', 'certificate', 'mess', 'id_card', 'miscellaneous', 'custom'
                     )),
    status           VARCHAR(20) NOT NULL DEFAULT 'active'
                     CHECK (status IN ('active', 'inactive')),
    taxable          BOOLEAN NOT NULL DEFAULT FALSE,
    tax_rate_bps     INTEGER NOT NULL DEFAULT 0,
    metadata         JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ,
    CHECK (tax_rate_bps >= 0 AND tax_rate_bps <= 10000),
    UNIQUE (tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_fee_heads_active_code
    ON fee_heads (tenant_id, lower(code))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_fee_heads_tenant_status
    ON fee_heads (tenant_id, status)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_fee_heads_tenant_category
    ON fee_heads (tenant_id, category)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS fee_structures (
    id                           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                    UUID NOT NULL REFERENCES tenants(id),
    academic_year_id             UUID NOT NULL,
    name                         VARCHAR(160) NOT NULL,
    code                         VARCHAR(60) NOT NULL,
    description                  TEXT NOT NULL DEFAULT '',
    billing_cycle                VARCHAR(20) NOT NULL DEFAULT 'one_time'
                                 CHECK (billing_cycle IN ('one_time', 'monthly', 'quarterly', 'term', 'yearly', 'custom')),
    status                       VARCHAR(20) NOT NULL DEFAULT 'draft'
                                 CHECK (status IN ('draft', 'active', 'inactive', 'archived')),
    currency                     CHAR(3) NOT NULL DEFAULT 'INR',
    allow_partial_payment        BOOLEAN NOT NULL DEFAULT FALSE,
    minimum_partial_amount_paise BIGINT NOT NULL DEFAULT 0,
    due_day                      INTEGER,
    metadata                     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at                   TIMESTAMPTZ,
    CHECK (minimum_partial_amount_paise >= 0),
    CHECK (due_day IS NULL OR (due_day >= 1 AND due_day <= 31)),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_fee_structures_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_fee_structures_active_code
    ON fee_structures (tenant_id, academic_year_id, lower(code))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_fee_structures_tenant_year_status
    ON fee_structures (tenant_id, academic_year_id, status)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS fee_structure_items (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL REFERENCES tenants(id),
    fee_structure_id         UUID NOT NULL,
    fee_head_id              UUID NOT NULL,
    name                     VARCHAR(160) NOT NULL DEFAULT '',
    description              TEXT NOT NULL DEFAULT '',
    amount_paise             BIGINT NOT NULL,
    tax_rate_bps             INTEGER NOT NULL DEFAULT 0,
    sort_order               INTEGER NOT NULL DEFAULT 0,
    optional                 BOOLEAN NOT NULL DEFAULT FALSE,
    metadata                 JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at               TIMESTAMPTZ,
    CHECK (amount_paise >= 0),
    CHECK (tax_rate_bps >= 0 AND tax_rate_bps <= 10000),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_fee_structure_items_structure
        FOREIGN KEY (tenant_id, fee_structure_id)
        REFERENCES fee_structures(tenant_id, id),
    CONSTRAINT fk_fee_structure_items_head
        FOREIGN KEY (tenant_id, fee_head_id)
        REFERENCES fee_heads(tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_fee_structure_items_structure
    ON fee_structure_items (tenant_id, fee_structure_id, sort_order)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_fee_structure_items_head
    ON fee_structure_items (tenant_id, fee_head_id)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS student_fee_assignments (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id),
    fee_structure_id   UUID NOT NULL,
    academic_year_id   UUID NOT NULL,
    assignment_type    VARCHAR(20) NOT NULL
                       CHECK (assignment_type IN ('class', 'section', 'student')),
    class_id           UUID,
    section_id         UUID,
    student_id         UUID,
    status             VARCHAR(20) NOT NULL DEFAULT 'active'
                       CHECK (status IN ('active', 'inactive', 'cancelled')),
    effective_from     DATE NOT NULL,
    effective_until    DATE,
    created_by         UUID REFERENCES users(id),
    metadata           JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at         TIMESTAMPTZ,
    CHECK (effective_until IS NULL OR effective_until >= effective_from),
    CHECK (
        (assignment_type = 'class' AND class_id IS NOT NULL AND section_id IS NULL AND student_id IS NULL) OR
        (assignment_type = 'section' AND class_id IS NOT NULL AND section_id IS NOT NULL AND student_id IS NULL) OR
        (assignment_type = 'student' AND student_id IS NOT NULL)
    ),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_student_fee_assignments_structure
        FOREIGN KEY (tenant_id, fee_structure_id)
        REFERENCES fee_structures(tenant_id, id),
    CONSTRAINT fk_student_fee_assignments_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id),
    CONSTRAINT fk_student_fee_assignments_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES classes(tenant_id, id),
    CONSTRAINT fk_student_fee_assignments_section
        FOREIGN KEY (tenant_id, section_id)
        REFERENCES sections(tenant_id, id),
    CONSTRAINT fk_student_fee_assignments_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES students(tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_student_fee_assignments_structure
    ON student_fee_assignments (tenant_id, fee_structure_id, status)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_student_fee_assignments_target
    ON student_fee_assignments (tenant_id, academic_year_id, assignment_type, class_id, section_id, student_id)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS concessions (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id),
    academic_year_id   UUID NOT NULL,
    student_id         UUID NOT NULL,
    fee_head_id        UUID,
    name               VARCHAR(160) NOT NULL,
    code               VARCHAR(60) NOT NULL DEFAULT '',
    concession_type    VARCHAR(20) NOT NULL
                       CHECK (concession_type IN ('fixed_amount', 'percentage')),
    amount_paise       BIGINT NOT NULL DEFAULT 0,
    percentage_bps     INTEGER NOT NULL DEFAULT 0,
    reason             TEXT NOT NULL DEFAULT '',
    status             VARCHAR(20) NOT NULL DEFAULT 'active'
                       CHECK (status IN ('active', 'inactive', 'expired')),
    starts_on          DATE NOT NULL,
    ends_on            DATE,
    metadata           JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at         TIMESTAMPTZ,
    CHECK (amount_paise >= 0),
    CHECK (percentage_bps >= 0 AND percentage_bps <= 10000),
    CHECK (
        (concession_type = 'fixed_amount' AND amount_paise > 0 AND percentage_bps = 0) OR
        (concession_type = 'percentage' AND percentage_bps > 0 AND amount_paise = 0)
    ),
    CHECK (ends_on IS NULL OR ends_on >= starts_on),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_concessions_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id),
    CONSTRAINT fk_concessions_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES students(tenant_id, id),
    CONSTRAINT fk_concessions_fee_head
        FOREIGN KEY (tenant_id, fee_head_id)
        REFERENCES fee_heads(tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_concessions_student_active
    ON concessions (tenant_id, student_id, academic_year_id, status, starts_on, ends_on)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_concessions_fee_head
    ON concessions (tenant_id, fee_head_id)
    WHERE fee_head_id IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS late_fee_rules (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id),
    fee_structure_id   UUID,
    fee_head_id        UUID,
    name               VARCHAR(160) NOT NULL,
    rule_type          VARCHAR(20) NOT NULL
                       CHECK (rule_type IN ('fixed', 'daily', 'weekly', 'monthly')),
    amount_paise       BIGINT NOT NULL,
    grace_days         INTEGER NOT NULL DEFAULT 0,
    max_amount_paise   BIGINT,
    status             VARCHAR(20) NOT NULL DEFAULT 'active'
                       CHECK (status IN ('active', 'inactive')),
    effective_from     DATE NOT NULL,
    effective_until    DATE,
    metadata           JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at         TIMESTAMPTZ,
    CHECK (amount_paise >= 0),
    CHECK (grace_days >= 0),
    CHECK (max_amount_paise IS NULL OR max_amount_paise >= 0),
    CHECK (effective_until IS NULL OR effective_until >= effective_from),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_late_fee_rules_structure
        FOREIGN KEY (tenant_id, fee_structure_id)
        REFERENCES fee_structures(tenant_id, id),
    CONSTRAINT fk_late_fee_rules_head
        FOREIGN KEY (tenant_id, fee_head_id)
        REFERENCES fee_heads(tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_late_fee_rules_tenant_status
    ON late_fee_rules (tenant_id, status, effective_from)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_late_fee_rules_structure
    ON late_fee_rules (tenant_id, fee_structure_id)
    WHERE fee_structure_id IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS invoice_counters (
    tenant_id        UUID NOT NULL REFERENCES tenants(id),
    academic_year_id UUID NOT NULL,
    prefix           VARCHAR(80) NOT NULL,
    next_number      BIGINT NOT NULL DEFAULT 1,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, academic_year_id),
    CONSTRAINT fk_invoice_counters_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id)
);

CREATE TABLE IF NOT EXISTS invoices (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL REFERENCES tenants(id),
    invoice_number           VARCHAR(80) NOT NULL,
    student_id               UUID NOT NULL,
    academic_year_id         UUID NOT NULL,
    class_id                 UUID NOT NULL,
    section_id               UUID NOT NULL,
    fee_structure_id         UUID,
    assignment_id            UUID,
    issue_date               DATE NOT NULL,
    due_date                 DATE NOT NULL,
    billing_period_start     DATE,
    billing_period_end       DATE,
    generation_key           VARCHAR(180) NOT NULL,
    status                   VARCHAR(20) NOT NULL DEFAULT 'issued'
                             CHECK (status IN ('draft', 'issued', 'partially_paid', 'paid', 'overdue', 'cancelled', 'void')),
    currency                 CHAR(3) NOT NULL DEFAULT 'INR',
    allow_partial_payment       BOOLEAN NOT NULL DEFAULT FALSE,
    minimum_partial_amount_paise BIGINT NOT NULL DEFAULT 0,
    subtotal_amount_paise    BIGINT NOT NULL DEFAULT 0,
    discount_amount_paise    BIGINT NOT NULL DEFAULT 0,
    fine_amount_paise        BIGINT NOT NULL DEFAULT 0,
    tax_amount_paise         BIGINT NOT NULL DEFAULT 0,
    total_amount_paise       BIGINT NOT NULL DEFAULT 0,
    paid_amount_paise        BIGINT NOT NULL DEFAULT 0,
    balance_amount_paise     BIGINT NOT NULL DEFAULT 0,
    generated_by             UUID REFERENCES users(id),
    metadata                 JSONB NOT NULL DEFAULT '{}'::jsonb,
    issued_at                TIMESTAMPTZ,
    cancelled_at             TIMESTAMPTZ,
    voided_at                TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (due_date >= issue_date),
    CHECK (billing_period_end IS NULL OR billing_period_start IS NOT NULL),
    CHECK (billing_period_start IS NULL OR billing_period_end IS NOT NULL),
    CHECK (billing_period_end IS NULL OR billing_period_end >= billing_period_start),
    CHECK (subtotal_amount_paise >= 0),
    CHECK (minimum_partial_amount_paise >= 0),
    CHECK (discount_amount_paise >= 0),
    CHECK (fine_amount_paise >= 0),
    CHECK (tax_amount_paise >= 0),
    CHECK (total_amount_paise >= 0),
    CHECK (paid_amount_paise >= 0),
    CHECK (balance_amount_paise >= 0),
    CHECK (subtotal_amount_paise - discount_amount_paise + fine_amount_paise + tax_amount_paise = total_amount_paise),
    CHECK (total_amount_paise - paid_amount_paise = balance_amount_paise),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_invoices_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES students(tenant_id, id),
    CONSTRAINT fk_invoices_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id),
    CONSTRAINT fk_invoices_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES classes(tenant_id, id),
    CONSTRAINT fk_invoices_section
        FOREIGN KEY (tenant_id, section_id)
        REFERENCES sections(tenant_id, id),
    CONSTRAINT fk_invoices_structure
        FOREIGN KEY (tenant_id, fee_structure_id)
        REFERENCES fee_structures(tenant_id, id),
    CONSTRAINT fk_invoices_assignment
        FOREIGN KEY (tenant_id, assignment_id)
        REFERENCES student_fee_assignments(tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_invoices_tenant_number
    ON invoices (tenant_id, invoice_number);

CREATE UNIQUE INDEX IF NOT EXISTS idx_invoices_generation_key_active
    ON invoices (tenant_id, generation_key)
    WHERE status NOT IN ('cancelled', 'void');

CREATE INDEX IF NOT EXISTS idx_invoices_student_status_due
    ON invoices (tenant_id, student_id, status, due_date DESC);

CREATE INDEX IF NOT EXISTS idx_invoices_tenant_status_due
    ON invoices (tenant_id, status, due_date DESC);

CREATE INDEX IF NOT EXISTS idx_invoices_assignment
    ON invoices (tenant_id, assignment_id, created_at DESC)
    WHERE assignment_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS invoice_items (
    id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                  UUID NOT NULL REFERENCES tenants(id),
    invoice_id                 UUID NOT NULL,
    fee_head_id                UUID NOT NULL,
    fee_structure_item_id      UUID,
    description                TEXT NOT NULL DEFAULT '',
    amount_paise               BIGINT NOT NULL,
    discount_amount_paise      BIGINT NOT NULL DEFAULT 0,
    fine_amount_paise          BIGINT NOT NULL DEFAULT 0,
    tax_amount_paise           BIGINT NOT NULL DEFAULT 0,
    total_amount_paise         BIGINT NOT NULL,
    sort_order                 INTEGER NOT NULL DEFAULT 0,
    metadata                   JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (amount_paise >= 0),
    CHECK (discount_amount_paise >= 0),
    CHECK (fine_amount_paise >= 0),
    CHECK (tax_amount_paise >= 0),
    CHECK (total_amount_paise >= 0),
    CHECK (amount_paise - discount_amount_paise + fine_amount_paise + tax_amount_paise = total_amount_paise),
    UNIQUE (tenant_id, id),
    CONSTRAINT fk_invoice_items_invoice
        FOREIGN KEY (tenant_id, invoice_id)
        REFERENCES invoices(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_invoice_items_head
        FOREIGN KEY (tenant_id, fee_head_id)
        REFERENCES fee_heads(tenant_id, id),
    CONSTRAINT fk_invoice_items_structure_item
        FOREIGN KEY (tenant_id, fee_structure_item_id)
        REFERENCES fee_structure_items(tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice
    ON invoice_items (tenant_id, invoice_id, sort_order);

CREATE INDEX IF NOT EXISTS idx_invoice_items_fee_head
    ON invoice_items (tenant_id, fee_head_id);

INSERT INTO permissions (code, name, category, description) VALUES
    ('fees.manage', 'Manage Fees', 'tenant', 'Create fee setup, assignments, and generated invoices')
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
JOIN permissions p ON p.code = 'fees.manage'
WHERE r.slug = 'admin'
ON CONFLICT DO NOTHING;
