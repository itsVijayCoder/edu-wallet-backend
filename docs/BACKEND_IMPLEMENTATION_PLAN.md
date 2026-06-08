# EduWallet Backend Implementation Plan

Date: 2026-06-08

## 1. Purpose

This document is the backend implementation guide for building EduWallet into a production-grade MVP.

Primary product source:

- `docs/EduWallet_Product_Plan.md` for detailed feature and workflow requirements.
- `docs/PRODUCT_PLAN.md` for strategic direction, risks, compliance, pilot metrics, and launch discipline.
- `docs/EduWallet_Product_Plan_Analysis.md` for prioritization and competitive differentiation.

The backend must stay fee-first. Do not expand into full ERP modules until fee collection, payment accuracy, receipts, reminders, and reports are reliable.

## 2. MVP Target

Build a multi-tenant Go backend that allows a school or coaching center to:

1. Create an institution tenant and branch.
2. Manage tenant-scoped users, roles, and permissions.
3. Import and manage students, guardians, classes, sections, and academic years.
4. Configure fee heads, fee structures, assignments, and generated dues.
5. Accept online payments through Razorpay and offline payments through cash, cheque, DD, or bank transfer.
6. Generate receipts and maintain a student ledger.
7. Send basic reminders through provider interfaces.
8. Provide dashboards, defaulter lists, reports, and CSV exports.
9. Preserve audit history for sensitive financial and administrative actions.

Chosen implementation defaults:

- Backend only.
- REST API under `/api/v1`.
- Shared PostgreSQL database with tenant-scoped tables.
- Redis for sessions, rate limiting, and lightweight coordination.
- Razorpay first, in test mode during development.
- Resend email already exists; SMS and WhatsApp start as provider interfaces.
- Money is stored as integer paise with `currency = 'INR'`.
- No card data is stored.
- UPI AutoPay, fee financing, advanced AI, Tally sync, and multi-gateway routing are V2 or later.

## 3. Current Backend Baseline

Existing architecture:

```text
Handler -> Service -> Repository -> PostgreSQL
```

Existing packages:

- `cmd/api`: application entrypoint and dependency wiring.
- `internal/config`: env-based configuration.
- `internal/database`: PostgreSQL, Redis, DBTX, transaction helper.
- `internal/apperror`: machine-readable application errors.
- `internal/dto`: request and response objects.
- `internal/handler`: Gin HTTP handlers.
- `internal/middleware`: auth, role guard, rate limit, logging, CORS, recovery, security headers.
- `internal/model`: domain models.
- `internal/repository`: repository interfaces and Postgres implementations.
- `internal/router`: route registration.
- `internal/service`: business logic.
- `pkg`: email, password hashing, JWT, logger.
- `tests`: unit tests, e2e tests, mocks, test utilities.

Important baseline finding:

- `go build ./cmd/api` succeeds.
- `go test -short -race -count=1 ./tests/unit/...` succeeds.
- `go test -short -race -count=1 ./...` currently fails in `tests/e2e` when Docker is unavailable because e2e `TestMain` starts containers even in short mode. Fix this before adding major features.

## 4. Engineering Rules

Follow these rules for every phase:

1. Preserve the existing layered architecture.
2. Keep DTOs separate from models.
3. Keep repository interfaces in `internal/repository/interfaces.go` unless a module becomes large enough to justify splitting.
4. Keep services responsible for business rules and transaction boundaries.
5. Keep handlers thin: bind, validate, call service, respond.
6. Every tenant-owned table must include `tenant_id`.
7. Every tenant-owned query must filter by `tenant_id`.
8. Use UUID primary keys.
9. Use `deleted_at` for soft delete where records are administrative master data.
10. Do not hard-delete financial ledger, payment, receipt, webhook, or audit records.
11. Use database constraints for invariants that must never be violated.
12. Use partial indexes for common soft-delete filters.
13. Index tenant filters, foreign keys, status filters, and date ranges used by reports.
14. Keep database transactions short. Never call external payment or notification providers inside a transaction.
15. Make webhooks and payment updates idempotent.
16. Add tests with each feature commit.
17. Keep commit history scoped by task or feature.

## 5. Phase And Commit Plan

### Phase 0: Baseline Stabilization

Goal: make the existing repo safe to build on.

Tasks:

- Add a short-mode guard to e2e tests so Docker containers are skipped when `testing.Short()` is true.
- Ensure `make test` works without Docker.
- Ensure e2e tests still run with `make test-e2e` when Docker is available.
- Add or update test documentation in `README.md` only if behavior changes.
- Run:
  - `go test -short -race -count=1 ./...`
  - `go build ./cmd/api`
  - `go vet ./...`

Acceptance criteria:

- Short tests do not require Docker.
- Unit test and build baseline is green.
- No product behavior changes.

Commit:

```text
test: stabilize short test baseline
```

### Phase 1: Tenant, Auth, RBAC, And Audit Foundation

Goal: introduce tenant isolation before adding business modules.

Database:

- Add `tenants`.
- Add `tenant_branches`.
- Add `tenant_memberships` or equivalent user-to-tenant relation.
- Add `permissions`.
- Add `role_permissions`.
- Update role model to support platform and tenant roles.
- Add `audit_logs`.
- Add required indexes:
  - active tenant slug/domain.
  - tenant membership by `tenant_id`, `user_id`.
  - audit log by `tenant_id`, `actor_user_id`, `entity_type`, `entity_id`, `created_at`.

Backend:

- Add tenant model, DTOs, repository, service, handler, and routes.
- Add branch model, DTOs, repository, service, handler, and routes.
- Add tenant-aware auth claims.
- Add tenant context middleware.
- Add tenant switch or tenant selection endpoint.
- Update role guard to support tenant-scoped permissions.
- Keep platform super admin access separate from institution admin access.
- Disable public registration in production unless `AUTH_PUBLIC_REGISTRATION_ENABLED=true`.
- Add audit logger service and call it from sensitive write operations.

API modules:

- `POST /api/v1/platform/tenants`
- `GET /api/v1/platform/tenants`
- `GET /api/v1/platform/tenants/:id`
- `PATCH /api/v1/platform/tenants/:id`
- `POST /api/v1/platform/tenants/:id/branches`
- `GET /api/v1/admin/tenant`
- `PATCH /api/v1/admin/tenant`
- `POST /api/v1/auth/select-tenant`

Acceptance criteria:

- A user can belong to one or more tenants.
- JWT access tokens carry enough tenant context for scoped requests.
- Platform super admin can manage tenants.
- Institution admin can only access their tenant.
- Cross-tenant access is rejected.
- Audit logs are created for tenant and user administrative writes.

Commit:

```text
feat: add tenant-scoped auth and RBAC foundation
```

### Phase 2: Academic, Student, Guardian, And Import Core

Goal: support onboarding school data.

Database:

- Add `academic_years`.
- Add `classes`.
- Add `sections`.
- Add `students`.
- Add `guardians`.
- Add `student_guardians`.
- Add `imports`.
- Add `import_errors`.

Student fields:

- admission number.
- first name, last name.
- class and section.
- roll number.
- academic year.
- status: `active`, `inactive`, `transferred`, `graduated`.
- category: `general`, `scholarship`, `staff_child`, `sibling`, `custom`.
- contact and address fields needed for fee operations.
- opening balance in paise.
- optional metadata JSON for future custom fields.

Guardian fields:

- name.
- relationship.
- phone.
- WhatsApp phone.
- email.
- preferred language.
- communication opt-in.
- address.

Backend:

- Add academic setup services.
- Add student and guardian CRUD.
- Add guardian linking and sibling support.
- Add CSV import template endpoint.
- Add CSV import preview endpoint.
- Add CSV import commit endpoint.
- Add duplicate detection by tenant plus admission number.
- Store import history and row-level errors.

API modules:

- `GET/POST/PATCH/DELETE /api/v1/admin/academic-years`
- `GET/POST/PATCH/DELETE /api/v1/admin/classes`
- `GET/POST/PATCH/DELETE /api/v1/admin/sections`
- `GET/POST/PATCH/DELETE /api/v1/admin/students`
- `GET/POST/PATCH/DELETE /api/v1/admin/guardians`
- `POST /api/v1/admin/imports/students/preview`
- `POST /api/v1/admin/imports/students/commit`
- `GET /api/v1/admin/imports`

Acceptance criteria:

- Admin can create academic structure.
- Admin can create and list students by class, section, status, and search text.
- Admin can link multiple guardians to a student.
- One guardian can be linked to multiple students.
- Import preview reports row-level validation errors before commit.
- Import commit is transactional.
- Cross-tenant access is rejected.

Commit:

```text
feat: add academic and student management
```

### Phase 3: Fee Setup And Billing Engine

Goal: generate accurate dues from configurable fee structures.

Database:

- Add `fee_heads`.
- Add `fee_structures`.
- Add `fee_structure_items`.
- Add `student_fee_assignments`.
- Add `invoices`.
- Add `invoice_items`.
- Add `discounts` or `concessions` with MVP-level fields.
- Add `late_fee_rules` as configuration-ready, even if automation is later.

Core rules:

- Store all monetary values as integer paise.
- Never calculate final payable amount from client-submitted totals.
- Support one-time, monthly, quarterly, term-wise, yearly, and custom due dates.
- Support full and partial payment settings at tenant or fee structure level.
- Keep invoice item breakdown by fee head.
- Track invoice status: `draft`, `issued`, `partially_paid`, `paid`, `overdue`, `cancelled`, `void`.
- Track amount fields:
  - `subtotal_amount_paise`
  - `discount_amount_paise`
  - `fine_amount_paise`
  - `tax_amount_paise`
  - `total_amount_paise`
  - `paid_amount_paise`
  - `balance_amount_paise`

Backend:

- Add fee head CRUD.
- Add fee structure CRUD.
- Add assignment by class, section, or individual student.
- Add due generation service.
- Add invoice status recalculation service.
- Add student ledger read model from invoices, payments, receipts, and adjustments.

API modules:

- `GET/POST/PATCH/DELETE /api/v1/admin/fee-heads`
- `GET/POST/PATCH/DELETE /api/v1/admin/fee-structures`
- `POST /api/v1/admin/fee-assignments`
- `POST /api/v1/admin/invoices/generate`
- `GET /api/v1/admin/invoices`
- `GET /api/v1/admin/invoices/:id`
- `GET /api/v1/admin/students/:id/ledger`
- `GET /api/v1/parent/children/:id/dues`

Acceptance criteria:

- Admin can create fee structure and generate dues for students.
- Invoice totals match item totals.
- Partial payment eligibility is explicit.
- Invoice status changes are deterministic.
- Ledger endpoint shows correct balances.
- Fee and invoice queries are tenant-scoped.

Commit:

```text
feat: add fee setup and invoice generation
```

### Phase 4: Payments, Webhooks, Receipts, And Ledger

Goal: collect money correctly and produce auditable receipts.

Database:

- Add `payment_attempts`.
- Add `payments`.
- Add `gateway_webhooks`.
- Add `receipts`.
- Add `receipt_series`.
- Add `offline_payment_references`.
- Add `payment_events` for live payment ticker and operational history.

Payment statuses:

- `created`
- `pending`
- `processing`
- `success`
- `failed`
- `expired`
- `cancelled`
- `refunded`
- `partially_refunded`
- `manually_verified`
- `reconciliation_mismatch`
- `webhook_pending`
- `settlement_pending`
- `settled`

Provider design:

- Add `PaymentProvider` interface.
- Add Razorpay implementation.
- Add local test provider or fake provider for unit and e2e tests.
- Verify checkout signatures.
- Verify webhook signatures.
- Store gateway event IDs and reject duplicate processing.
- Do not store card data.

Receipt design:

- Receipt numbers are tenant scoped and academic-year or branch aware.
- Receipt creation must be idempotent for a successful payment.
- Receipt cancellation creates a cancellation record and audit log.
- PDF generation should use a provider interface so the renderer can change later.

Backend:

- Add order creation for selected invoices.
- Add payment attempt creation.
- Add Razorpay webhook endpoint.
- Add webhook processing service.
- Add offline payment entry.
- Add cheque/DD lifecycle fields.
- Add receipt generation.
- Add receipt download endpoint.
- Add payment event recording for live ticker.

API modules:

- `POST /api/v1/parent/payments/orders`
- `POST /api/v1/parent/payments/verify`
- `POST /api/v1/webhooks/razorpay`
- `POST /api/v1/admin/offline-payments`
- `GET /api/v1/admin/payments`
- `GET /api/v1/admin/payments/:id`
- `GET /api/v1/admin/receipts`
- `GET /api/v1/admin/receipts/:id`
- `GET /api/v1/admin/receipts/:id/download`
- `GET /api/v1/parent/receipts`
- `GET /api/v1/parent/receipts/:id/download`
- `GET /api/v1/admin/payment-events`

Acceptance criteria:

- Parent can create a Razorpay order for unpaid invoice balance.
- Webhook success updates payment and invoice balance exactly once.
- Duplicate webhook replay does not create duplicate receipts or overpay invoices.
- Failed payment records a failed attempt without marking invoice paid.
- Offline payment updates invoice balances and creates receipt.
- Receipt number is unique per tenant series.
- All payment and receipt writes create audit logs.

Commit:

```text
feat: add payments receipts and ledger
```

### Phase 5: Reminders, Notifications, Reports, And Exports

Goal: support daily finance workflows.

Database:

- Add `reminder_templates`.
- Add `reminder_rules`.
- Add `reminder_logs`.
- Add `notification_logs`.
- Add `jobs` or lightweight queue table if Redis-only processing is not enough.
- Add `export_jobs`.

Notification design:

- Add `NotificationProvider` interface.
- Keep provider-specific logic outside services.
- Email uses existing Resend client.
- SMS and WhatsApp are adapters with no hard dependency until vendor selection.
- Store delivery attempts and provider responses where safe.

Worker design:

- Add a background worker entrypoint or mode.
- Use short transactions.
- For Postgres-backed jobs, use `FOR UPDATE SKIP LOCKED`.
- Make job execution idempotent.
- Add retries with max attempts and last error.

Reports:

- Dashboard summary.
- Daily and date-range collection report.
- Defaulter report.
- Class/section-wise due report.
- Fee-head-wise collection report.
- Payment-mode report.
- Offline payment report.
- Receipt register.
- Export CSV for MVP.

API modules:

- `GET/POST/PATCH/DELETE /api/v1/admin/reminder-templates`
- `GET/POST/PATCH/DELETE /api/v1/admin/reminder-rules`
- `POST /api/v1/admin/reminders/send`
- `GET /api/v1/admin/reminder-logs`
- `GET /api/v1/admin/dashboard`
- `GET /api/v1/admin/reports/collections`
- `GET /api/v1/admin/reports/defaulters`
- `GET /api/v1/admin/reports/dues`
- `GET /api/v1/admin/reports/payment-methods`
- `POST /api/v1/admin/exports`
- `GET /api/v1/admin/exports/:id`
- `GET /api/v1/admin/exports/:id/download`

Acceptance criteria:

- Admin can see collection and due summaries.
- Admin can generate defaulter list.
- Admin can export collection and defaulter reports as CSV.
- Reminder logs show attempted channel, recipient, status, and invoice.
- Worker can safely retry failed reminder jobs.
- Reports remain tenant-scoped and indexed.

Commit:

```text
feat: add reminders reports and exports
```

### Phase 6: Production Hardening

Goal: prepare MVP for pilot institutions.

Security:

- Validate all required production env vars.
- Add strict CORS production config.
- Add request size limits for imports and webhooks.
- Add rate limits for auth, OTP-ready endpoints, payment order creation, and webhook endpoint if appropriate.
- Ensure payment webhook raw body verification is correct.
- Ensure no secrets are logged.
- Ensure audit logs cover sensitive write operations.

Reliability:

- Add graceful worker shutdown.
- Add provider timeouts.
- Add retry strategy for external providers.
- Add payment reconciliation-ready fields.
- Add admin tools for payment lookup.

Operations:

- Update `README.md`.
- Add API usage examples for key flows.
- Add migration and rollback notes.
- Add pilot seed data command or fixture.
- Add launch checklist.

Acceptance criteria:

- `go test -short -race -count=1 ./...` passes.
- E2E tests pass when Docker is available.
- `go vet ./...` passes.
- `golangci-lint run ./...` passes.
- Production build succeeds.
- Webhook replay test passes.
- 500-student CSV import test passes.
- Cross-tenant isolation tests pass.

Commit:

```text
chore: harden backend for MVP launch
```

## 6. Suggested Final MVP Data Model

Core platform:

- `tenants`
- `tenant_branches`
- `users`
- `roles`
- `permissions`
- `role_permissions`
- `tenant_memberships`
- `audit_logs`

Academic and people:

- `academic_years`
- `classes`
- `sections`
- `students`
- `guardians`
- `student_guardians`
- `imports`
- `import_errors`

Fees and billing:

- `fee_heads`
- `fee_structures`
- `fee_structure_items`
- `student_fee_assignments`
- `invoices`
- `invoice_items`
- `discounts`
- `concessions`
- `late_fee_rules`

Payments and receipts:

- `payment_attempts`
- `payments`
- `gateway_webhooks`
- `receipts`
- `receipt_series`
- `payment_events`
- `refunds` for V2-ready schema if not implemented in MVP behavior.
- `settlements` for V2-ready schema if not implemented in MVP behavior.

Communication and exports:

- `reminder_templates`
- `reminder_rules`
- `reminder_logs`
- `notification_logs`
- `jobs`
- `export_jobs`

SaaS:

- `subscription_plans`
- `tenant_subscriptions`
- `tenant_usage_snapshots`

## 7. API Shape Guidelines

Use the existing response envelope:

```json
{
  "success": true,
  "request_id": "request-id",
  "data": {}
}
```

Use paginated responses for list endpoints:

```json
{
  "success": true,
  "data": [],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

For large report tables, add cursor pagination after MVP stabilization. The current offset pagination can remain for admin CRUD where row counts are limited, but reports and ledgers should be designed to support cursor pagination.

## 8. Error Code Guidelines

Add domain-specific errors to `internal/apperror/apperror.go`.

Examples:

```text
TENANT_NOT_FOUND
TENANT_ACCESS_DENIED
STUDENT_NOT_FOUND
GUARDIAN_NOT_FOUND
FEE_STRUCTURE_NOT_FOUND
INVOICE_NOT_FOUND
INVOICE_ALREADY_PAID
PAYMENT_AMOUNT_INVALID
PAYMENT_PROVIDER_ERROR
PAYMENT_SIGNATURE_INVALID
WEBHOOK_DUPLICATE
RECEIPT_NOT_FOUND
IMPORT_VALIDATION_FAILED
REPORT_TOO_LARGE
```

Handlers must use `HandleError(c, err)` for application errors.

## 9. Testing Strategy

Unit tests:

- Service business rules.
- Invoice total and balance calculations.
- Partial payment validation.
- Tenant access checks.
- Role and permission checks.
- Provider signature verification.
- Webhook idempotency.
- Receipt numbering.
- Import validation.

Repository tests:

- Tenant-scoped filters.
- Unique constraints.
- Soft delete behavior.
- Invoice generation queries.
- Payment update transactions.
- Report query correctness.

E2E tests:

- Tenant onboarding.
- User login and tenant selection.
- Student import preview and commit.
- Fee structure creation.
- Invoice generation.
- Razorpay webhook success.
- Razorpay webhook replay.
- Payment failure.
- Offline payment.
- Receipt download.
- Dashboard summary.
- Defaulter export.
- Cross-tenant denial.

Manual pilot checks:

- Import at least 500 sample students.
- Generate monthly and term dues.
- Complete Razorpay test payments.
- Replay webhooks.
- Record offline cash and cheque payments.
- Download receipts.
- Export reports.
- Validate tenant isolation.

## 10. Clean Git Workflow

Use one branch for the full backend MVP:

```text
feature/backend-production-mvp
```

Commit rules:

- One commit per phase or independently reviewable feature.
- Do not mix formatting-only changes with behavior changes.
- Do not commit generated binaries, local `.env`, coverage reports, or temporary files.
- Keep migrations and code that depends on them in the same commit.
- Include tests in the same commit as the feature.
- Run the relevant test gate before each commit.

Recommended commit sequence:

```text
test: stabilize short test baseline
feat: add tenant-scoped auth and RBAC foundation
feat: add academic and student management
feat: add fee setup and invoice generation
feat: add payments receipts and ledger
feat: add reminders reports and exports
chore: harden backend for MVP launch
docs: document backend MVP implementation
```

## 11. Launch Checklist

Before pilot launch:

1. Create one demo tenant with realistic Indian school data.
2. Test online payments in Razorpay test mode.
3. Test webhook replay and duplicate protection.
4. Import at least 500 sample students through CSV.
5. Generate monthly and term dues.
6. Send test reminders.
7. Download receipts.
8. Export collection and defaulter reports.
9. Validate tenant isolation through automated tests.
10. Validate all production env vars.
11. Confirm no secrets or payment-sensitive data are logged.
12. Prepare admin onboarding notes.

## 12. Deferred Items

Do not build these in the first MVP unless explicitly reprioritized:

- Full mobile app.
- WhatsApp mini-app payment flow.
- UPI AutoPay or eNACH mandate execution.
- Fee financing provider integration.
- AI defaulter prediction.
- Tally or Zoho Books sync.
- Multi-gateway routing.
- Advanced settlement reconciliation.
- White-label custom domains.
- No-due certificates.
- Full ERP modules such as attendance, exams, homework, payroll, or transport operations.

## 13. Definition Of Done

The backend MVP is done when:

- A tenant can be onboarded.
- Students and guardians can be imported.
- Fee structures can generate dues.
- Parents can pay online through Razorpay test mode.
- Admins can record offline payments.
- Receipts are generated exactly once per successful payment.
- Dashboards and reports show correct totals.
- Reminders can be sent through provider interfaces.
- Audit logs capture sensitive actions.
- Cross-tenant data access is blocked.
- Short tests, build, vet, lint, and Docker-backed e2e pass in the appropriate environments.
