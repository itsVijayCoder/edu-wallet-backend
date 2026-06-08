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

1. Register or create a user.
2. Create a tenant as `super_admin`.
3. Call `POST /api/v1/auth/select-tenant` to get a tenant-scoped JWT.
4. Create academic year, class, section, students, and guardians.
5. Create fee heads, fee structures, assignments, and invoices.
6. Parents view dues and create payment orders.
7. Payments are verified or received through Razorpay webhooks.
8. Receipts, ledgers, dashboard, reports, reminders, and exports read from the same tenant-scoped records.
