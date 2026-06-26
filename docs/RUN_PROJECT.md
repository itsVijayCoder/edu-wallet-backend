# Run EduWallet Backend

This guide starts the backend locally with Postgres and Redis, applies migrations, opens the Swagger docs, and runs the API test suites.

## Prerequisites

- Go `1.25.x`
- Docker Desktop or Docker Engine
- `make`
- `migrate` CLI for local database migrations
- Optional: `air` for hot reload, `golangci-lint` for linting

## 1. Configure Environment

```bash
cp .env.example .env
```

For local development, the defaults are enough:

- `APP_ENV=development`
- `APP_PORT=8080`
- `DB_HOST=localhost`
- `REDIS_HOST=localhost`
- `PAYMENT_PROVIDER=fake`

For production, use real secrets and HTTPS values. The app will fail fast when production uses wildcard CORS, disabled DB SSL, placeholder Resend sender settings, missing Razorpay secrets, or reused JWT secrets.

For deployed or already-existing databases, bootstrap or rotate the super admin with:

```env
SUPER_ADMIN_BOOTSTRAP_ENABLED=true
SUPER_ADMIN_EMAIL=admin@eduwallet.in
SUPER_ADMIN_PASSWORD=<strong-password>
SUPER_ADMIN_FIRST_NAME=EduWallet
SUPER_ADMIN_LAST_NAME=Owner
```

Start the API once with those values, confirm login, then set `SUPER_ADMIN_BOOTSTRAP_ENABLED=false` and restart/redeploy.

## 2. Start Dependencies

```bash
make docker-up
```

This starts:

- Postgres on `localhost:5432`
- Redis on `localhost:6379`
- The API container if Docker Compose builds the full stack

For local Go development, you can leave Postgres and Redis running and start the API from source in the next step.

## 3. Run Migrations

```bash
make migrate-up
```

Migrations also seed the product owner/developer account:

```text
Email: admin@eduwallet.in
Password: password
Role: super_admin
```

Use this login to create school/college admins and tenants. Rotate the password before using a shared or production database.

Useful migration checks:

```bash
make migrate-version
make migrate-down
```

Only use `make migrate-down` to roll back the latest migration. Financial tables are immutable operational records; do not manually delete payments, receipts, gateway events, ledger rows, or audit logs.

## 4. Start The API

Hot reload:

```bash
make dev
```

Build and run:

```bash
make build
./bin/eduwallet
```

The API runs at:

```text
http://localhost:8080
```

Health checks:

```bash
curl http://localhost:8080/api/v1/healthz
curl http://localhost:8080/api/v1/readyz
```

## 5. Open Swagger Docs

Browser UI:

```text
http://localhost:8080/api/v1/docs
```

Machine-readable specs:

```text
http://localhost:8080/api/v1/docs/openapi.json
http://localhost:8080/api/v1/docs/swagger.json
```

Regenerate checked-in docs:

```bash
make swagger
```

The generated files are written to `docs/swagger/openapi.json` and `docs/swagger/swagger.json`.

## 6. Run Tests

Fast tests:

```bash
make test
```

Full Docker-backed API tests:

```bash
make test-e2e
```

Quality checks:

```bash
make vet
make lint
make build
```

## 7. Run Worker Mode

The same binary can process reminder retry jobs:

```bash
APP_MODE=worker WORKER_POLL_INTERVAL=5s ./bin/eduwallet
```

Run the API and worker as separate processes in production.

## Local API Flow

For a non-technical school tester walkthrough, open `docs/SCHOOL_TEST_JOURNEY.md`.

High-level flow:

1. Login as `admin@eduwallet.in` with the local seeded password `password`, or with the deployed password set through `SUPER_ADMIN_PASSWORD`.
2. Create a school/college admin user.
3. Create a school/college tenant with that admin as `owner_user_id`.
4. Login as the school/college admin.
5. Call `POST /api/v1/auth/select-tenant` to get a tenant-scoped JWT.
6. Create tenant users with `POST /api/v1/admin/tenant/users` when needed.
7. Create academic year, class, section, guardians, and students.
8. Create fee heads, fee structures, assignments, and invoices.
9. View dues, record payment, download receipt, and check ledger.
10. Test reminders, dashboard, reports, and CSV exports.
