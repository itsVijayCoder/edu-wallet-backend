# eduwallet

My API

## Quick Start

```bash
cp .env.example .env     # Configure your environment
make docker-up           # Start Postgres + Redis
make migrate-up          # Run database migrations
make dev                 # Start with hot reload
```

API is available at `http://localhost:8080`

## Tech Stack

- **Language:** Go 1.25
- **Framework:** Gin
- **Database:** PostgreSQL 16 (pgx/v5, raw SQL, DBTX transactions)
- **Cache:** Redis 7 (sessions, rate limiting)
- **Auth:** JWT (access + refresh tokens, bcrypt)
- **Migrations:** golang-migrate
- **Testing:** testify + testcontainers
- **CI:** GitHub Actions

## Project Structure

```
.
├── cmd/api/main.go              # Entry point, DI wiring
├── internal/
│   ├── config/                  # Environment-based configuration
│   ├── database/                # Postgres pool, Redis client, DBTX interface
│   ├── apperror/                # Structured error types with machine-readable codes
│   ├── dto/                     # Request/response data transfer objects
│   ├── handler/                 # HTTP handlers (bind, validate, respond)
│   ├── middleware/              # Auth, CORS, logger, rate limit, recovery, etc.
│   ├── model/                   # Domain models
│   ├── repository/              # Data access interfaces
│   │   └── postgres/            # PostgreSQL implementations
│   ├── router/                  # Route definitions with middleware chains
│   └── service/                 # Business logic
├── pkg/                         # Shared packages
│   ├── email/                   # Resend email client
│   ├── hasher/                  # Bcrypt password hashing
│   ├── jwt/                     # Token manager (access + refresh)
│   └── logger/                  # Structured slog logger
├── migrations/                  # SQL migration files
├── tests/                       # Test suites
│   ├── unit/                    # Unit tests with mocks
│   ├── e2e/                     # E2E tests with testcontainers
│   └── mocks/                   # Hand-written mock implementations
├── docs/                        # API documentation
│   └── swagger/                 # Generated Swagger specs
├── Makefile                     # All project commands
├── Dockerfile                   # Multi-stage production build
└── docker-compose.yml           # Local dev services (Postgres + Redis)
```

## API Endpoints

### Auth

| Method | Endpoint                          | Description                |
|--------|-----------------------------------|----------------------------|
| POST   | `/api/v1/auth/register`           | Register new user          |
| POST   | `/api/v1/auth/login`              | Login (returns tokens)     |
| POST   | `/api/v1/auth/refresh`            | Refresh access token       |
| POST   | `/api/v1/auth/select-tenant`      | Select tenant context      |
| POST   | `/api/v1/auth/logout`             | Logout (requires auth)     |
| POST   | `/api/v1/auth/forgot-password`    | Request password reset     |
| POST   | `/api/v1/auth/reset-password`     | Reset password with token  |

### Platform (requires super_admin role)

| Method | Endpoint                                  | Description              |
|--------|-------------------------------------------|--------------------------|
| POST   | `/api/v1/platform/tenants`                | Create tenant            |
| GET    | `/api/v1/platform/tenants`                | List tenants             |
| GET    | `/api/v1/platform/tenants/:id`            | Get tenant by ID         |
| PATCH  | `/api/v1/platform/tenants/:id`            | Update tenant            |
| POST   | `/api/v1/platform/tenants/:id/branches`   | Create tenant branch     |

### Tenant Admin (requires selected tenant token)

| Method | Endpoint                    | Description              |
|--------|-----------------------------|--------------------------|
| GET    | `/api/v1/admin/tenant`      | Get current tenant       |
| PATCH  | `/api/v1/admin/tenant`      | Update current tenant    |

### Academic Setup (requires selected tenant token + `academic.manage`)

| Method | Endpoint                              | Description                |
|--------|---------------------------------------|----------------------------|
| POST   | `/api/v1/admin/academic-years`        | Create academic year       |
| GET    | `/api/v1/admin/academic-years`        | List academic years        |
| GET    | `/api/v1/admin/academic-years/:id`    | Get academic year by ID    |
| PATCH  | `/api/v1/admin/academic-years/:id`    | Update academic year       |
| DELETE | `/api/v1/admin/academic-years/:id`    | Soft delete academic year  |
| POST   | `/api/v1/admin/classes`               | Create class               |
| GET    | `/api/v1/admin/classes`               | List classes               |
| GET    | `/api/v1/admin/classes/:id`           | Get class by ID            |
| PATCH  | `/api/v1/admin/classes/:id`           | Update class               |
| DELETE | `/api/v1/admin/classes/:id`           | Soft delete class          |
| POST   | `/api/v1/admin/sections`              | Create section             |
| GET    | `/api/v1/admin/sections`              | List sections              |
| GET    | `/api/v1/admin/sections/:id`          | Get section by ID          |
| PATCH  | `/api/v1/admin/sections/:id`          | Update section             |
| DELETE | `/api/v1/admin/sections/:id`          | Soft delete section        |

### Students And Guardians (requires selected tenant token)

| Method | Endpoint                                           | Permission          | Description               |
|--------|----------------------------------------------------|---------------------|---------------------------|
| POST   | `/api/v1/admin/students`                           | `students.manage`   | Create student            |
| GET    | `/api/v1/admin/students`                           | `students.manage`   | List/search students      |
| GET    | `/api/v1/admin/students/:id`                       | `students.manage`   | Get student by ID         |
| PATCH  | `/api/v1/admin/students/:id`                       | `students.manage`   | Update student            |
| DELETE | `/api/v1/admin/students/:id`                       | `students.manage`   | Soft delete student       |
| POST   | `/api/v1/admin/students/:id/guardians`             | `students.manage`   | Link guardian to student  |
| DELETE | `/api/v1/admin/students/:id/guardians/:guardian_id`| `students.manage`   | Unlink guardian           |
| POST   | `/api/v1/admin/guardians`                          | `guardians.manage`  | Create guardian           |
| GET    | `/api/v1/admin/guardians`                          | `guardians.manage`  | List/search guardians     |
| GET    | `/api/v1/admin/guardians/:id`                      | `guardians.manage`  | Get guardian by ID        |
| PATCH  | `/api/v1/admin/guardians/:id`                      | `guardians.manage`  | Update guardian           |
| DELETE | `/api/v1/admin/guardians/:id`                      | `guardians.manage`  | Soft delete guardian      |

Student list supports `academic_year_id`, `class_id`, `section_id`, `status`, and `search` filters.

### Student Imports (requires selected tenant token + `imports.manage`)

| Method | Endpoint                                      | Description                    |
|--------|-----------------------------------------------|--------------------------------|
| GET    | `/api/v1/admin/imports`                       | List import history            |
| GET    | `/api/v1/admin/imports/students/template`     | Download student CSV template  |
| POST   | `/api/v1/admin/imports/students/preview`      | Validate and store preview     |
| POST   | `/api/v1/admin/imports/students/commit`       | Commit a clean preview         |

Student import preview accepts multipart `file`, raw `text/csv`, or JSON `{ "filename": "students.csv", "csv": "..." }`. Commit accepts `{ "import_id": "..." }` and runs transactionally.

### Fees, Billing, And Ledgers (requires selected tenant token + `fees.manage`)

| Method | Endpoint                                  | Description                  |
|--------|-------------------------------------------|------------------------------|
| POST   | `/api/v1/admin/fee-heads`                 | Create fee head              |
| GET    | `/api/v1/admin/fee-heads`                 | List/search fee heads        |
| GET    | `/api/v1/admin/fee-heads/:id`             | Get fee head by ID           |
| PATCH  | `/api/v1/admin/fee-heads/:id`             | Update fee head              |
| DELETE | `/api/v1/admin/fee-heads/:id`             | Soft delete fee head         |
| POST   | `/api/v1/admin/fee-structures`            | Create fee structure + items |
| GET    | `/api/v1/admin/fee-structures`            | List fee structures          |
| GET    | `/api/v1/admin/fee-structures/:id`        | Get fee structure by ID      |
| PATCH  | `/api/v1/admin/fee-structures/:id`        | Update fee structure         |
| DELETE | `/api/v1/admin/fee-structures/:id`        | Soft delete fee structure    |
| POST   | `/api/v1/admin/fee-assignments`           | Assign fees by class, section, or student |
| POST   | `/api/v1/admin/invoices/generate`         | Generate dues from an assignment |
| GET    | `/api/v1/admin/invoices`                  | List invoices                |
| GET    | `/api/v1/admin/invoices/:id`              | Get invoice with item breakdown |
| GET    | `/api/v1/admin/students/:id/ledger`       | Read student ledger          |

Invoice generation accepts an `assignment_id`, optional `student_ids`, `issue_date`, `due_date`, and recurring `billing_period_start` / `billing_period_end`. Totals are calculated server-side from fee structure items and active concessions; client-submitted totals are ignored. Repeating the same assignment/student/period generation skips existing invoices through an idempotent generation key.

Invoices store explicit partial-payment rules: `allow_partial_payment` and `minimum_partial_amount_paise`. Student ledger includes opening balance, generated invoices, successful online payments, manually verified offline payments, and receipt references.

### Payments, Webhooks, And Receipts

Parent payment endpoints require auth + selected tenant token. Admin finance endpoints require selected tenant token + `payments.manage`.

| Method | Endpoint                                  | Description                         |
|--------|-------------------------------------------|-------------------------------------|
| POST   | `/api/v1/parent/payments/orders`          | Create provider order for invoices  |
| POST   | `/api/v1/parent/payments/verify`          | Verify checkout signature and apply payment |
| POST   | `/api/v1/webhooks/razorpay`               | Process signed Razorpay webhook     |
| POST   | `/api/v1/admin/offline-payments`          | Record cleared offline payment      |
| GET    | `/api/v1/admin/payments`                  | List payments                       |
| GET    | `/api/v1/admin/payments/:id`              | Get payment with allocations        |
| GET    | `/api/v1/admin/receipts`                  | List receipts                       |
| GET    | `/api/v1/admin/receipts/:id`              | Get receipt                         |
| GET    | `/api/v1/admin/receipts/:id/download`     | Download receipt PDF                |
| GET    | `/api/v1/parent/receipts`                 | List parent receipts                |
| GET    | `/api/v1/parent/receipts/:id/download`    | Download parent receipt PDF         |
| GET    | `/api/v1/admin/payment-events`            | List payment ticker events          |

Order creation accepts `student_id`, `invoice_ids`, optional `amount_paise`, and optional `idempotency_key`. If `amount_paise` is omitted, the order covers the selected invoice balances. Partial payment is allowed only for one invoice at a time and must satisfy the invoice partial-payment rule.

Webhook and payment verification paths are idempotent by provider event/payment IDs. Successful payments atomically update invoice balances, create payment allocations, generate one receipt per payment, and record payment events/audit logs.

### Parent Dues (requires auth + selected tenant token)

| Method | Endpoint                              | Description             |
|--------|---------------------------------------|-------------------------|
| GET    | `/api/v1/parent/children/:id/dues`    | View unpaid child dues  |

### Admin (requires super_admin or admin role)

| Method | Endpoint                    | Description              |
|--------|-----------------------------|--------------------------|
| POST   | `/api/v1/admin/users`       | Create user              |
| GET    | `/api/v1/admin/users`       | List users (paginated)   |
| GET    | `/api/v1/admin/users/:id`   | Get user by ID           |
| PUT    | `/api/v1/admin/users/:id`   | Update user              |
| DELETE | `/api/v1/admin/users/:id`   | Soft delete user         |

### Health

| Method | Endpoint            | Description                          |
|--------|---------------------|--------------------------------------|
| GET    | `/api/v1/healthz`   | Liveness probe                       |
| GET    | `/api/v1/readyz`    | Readiness probe (dependency status)  |

## Adding Your First Entity

Use the existing User implementation as a reference. For example, to add a `Product` entity:

1. **Model** - Create `internal/model/product.go` with your struct (UUID primary key, `deleted_at` for soft delete)
2. **DTOs** - Create `internal/dto/product.go` with `CreateProductRequest`, `UpdateProductRequest`, and `ProductResponse`
3. **Error codes** - Add `ErrProductNotFound`, etc. to `internal/apperror/apperror.go`
4. **Repository interface** - Add `ProductRepository` to `internal/repository/interfaces.go`
5. **Repository implementation** - Create `internal/repository/postgres/product.go` (accepts `database.DBTX`)
6. **Service interface** - Add `ProductService` to `internal/service/interfaces.go`
7. **Service implementation** - Create `internal/service/product.go`
8. **Handler** - Create `internal/handler/product.go`
9. **Routes** - Register routes in `internal/router/router.go` (find the `ADD YOUR ROUTES HERE` comment)
10. **Wire DI** - Instantiate repo, service, and handler in `cmd/api/main.go` (find the `ADD YOUR` markers)
11. **Migration** - Run `make migrate-create name=create_products` and write your SQL

## Configuration

Copy `.env.example` to `.env` and adjust values. Key settings:

| Variable             | Default          | Description                     |
|----------------------|------------------|---------------------------------|
| `APP_ENV`            | `development`    | Environment (development/production) |
| `APP_PORT`           | `8080`           | HTTP server port                |
| `DB_HOST`            | `localhost`      | PostgreSQL host                 |
| `DB_PORT`            | `5432`           | PostgreSQL port                 |
| `REDIS_HOST`         | `localhost`      | Redis host                      |
| `JWT_ACCESS_SECRET`  | -                | JWT signing key (generate with `openssl rand -base64 48`) |
| `JWT_REFRESH_SECRET` | -                | Refresh token signing key       |
| `AUTH_PUBLIC_REGISTRATION_ENABLED` | `false` | Enables public registration in production |
| `RESEND_API_KEY`     | -                | Resend API key (optional, for emails) |

## Testing

```bash
make test        # Short tests with race detector; skips Docker-backed e2e containers
make test-e2e    # E2E tests with real Postgres + Redis (testcontainers)
make test-all    # Run everything
make coverage    # Generate HTML coverage report
```

## Deployment

### Docker

```bash
make docker-up           # Start all services
make docker-logs         # Tail logs
make docker-down         # Stop all services
```

### Build from Source

```bash
make build               # Produces bin/eduwallet
./bin/eduwallet
```

### Production Upgrade

```bash
make upgrade             # git pull, migrate, build, restart
```

## Available Make Commands

Run `make help` to see all commands:

| Command              | Description                        |
|----------------------|------------------------------------|
| `make dev`           | Hot reload development (air)       |
| `make build`         | Build production binary            |
| `make run`           | Build and run                      |
| `make test`          | Unit tests (short, race)           |
| `make test-e2e`      | E2E tests with testcontainers      |
| `make coverage`      | HTML coverage report               |
| `make lint`          | Run golangci-lint                  |
| `make fmt`           | Format code (gofmt + goimports)    |
| `make migrate-up`    | Apply pending migrations           |
| `make migrate-down`  | Roll back last migration           |
| `make migrate-create`| New migration (`name=X`)           |
| `make docker-up`     | Start Postgres + Redis             |
| `make docker-down`   | Stop all services                  |
| `make swagger`       | Generate Swagger docs              |
| `make upgrade`       | Pull, migrate, build, restart      |

## License

MIT
