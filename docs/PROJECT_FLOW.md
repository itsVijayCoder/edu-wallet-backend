# EduWallet Project Flow

Use this document to understand how a request moves through the backend and how the school-fee product workflow is modeled.

For a non-technical school tester path through Swagger, use `docs/SCHOOL_TEST_JOURNEY.md`.

## Architecture Image

![EduWallet backend flow](project-flow.svg)

## Architecture Mindmap

```mermaid
mindmap
  root((EduWallet Backend))
    Entry points
      cmd/api/main.go
      APP_MODE=api
      APP_MODE=worker
    HTTP layer
      internal/router
      middleware
        request id
        security headers
        CORS
        auth
        tenant guard
        role and permission guards
        rate limits
        body limits
      handlers
        bind request
        validate DTO
        call service
        envelope response
    Domain layer
      auth service
      tenant service
      academic service
      billing service
      payment service
      operations service
    Data layer
      repositories
      DBTX transactions
      migrations
      Postgres
      Redis
    External providers
      Razorpay
      Resend
      PDF receipt renderer
    Tests and docs
      unit tests
      e2e tests
      OpenAPI catalog
      Swagger UI
```

## Request Flow

```mermaid
flowchart LR
  Client[Admin app or parent app] --> Router[router.New]
  Router --> Middleware[Middleware chain]
  Middleware --> Handler[HTTP handler]
  Handler --> DTO[Bind and validate DTO]
  DTO --> Service[Service method]
  Service --> Repo[Repository interface]
  Repo --> DB[(Postgres)]
  Service --> Redis[(Redis)]
  Service --> Provider[Email/payment/PDF provider]
  Service --> Handler
  Handler --> Response[APIResponse envelope]
  Response --> Client
```

## Product Workflow

```mermaid
flowchart TD
  A[User login or register] --> B[Select tenant]
  B --> C[Create academic year, class, section]
  C --> D[Create/import students and guardians]
  D --> E[Create fee heads and fee structures]
  E --> F[Assign fee structure]
  F --> G[Generate invoices]
  G --> H[Parent views dues]
  H --> I[Create payment order]
  I --> J[Verify payment or process Razorpay webhook]
  J --> K[Apply allocations transactionally]
  K --> L[Create receipt and ledger events]
  L --> M[Dashboard, reports, reminders, exports]
```

## Layer Responsibilities

| Layer | Location | Responsibility |
|-------|----------|----------------|
| Entry point | `cmd/api/main.go` | Load config, connect Postgres/Redis, wire repositories, services, handlers, and router. |
| Router | `internal/router/router.go` | Own route paths and attach middleware for auth, tenant context, permissions, limits, and rate limits. |
| Handler | `internal/handler` | Bind JSON/form/query/path data, validate DTOs, get actor/tenant context, call services, format responses. |
| Service | `internal/service` | Enforce business rules, execute transactions, coordinate repositories and providers. |
| Repository | `internal/repository/postgres` | Run SQL through `database.DBTX`; no HTTP or provider logic belongs here. |
| Model/DTO | `internal/model`, `internal/dto` | Keep database/domain shape separate from API request and response shape. |
| Migration | `migrations` | Own schema changes and rollback scripts. |
| Tests | `tests`, `internal/*/*_test.go` | Unit tests for narrow logic and e2e tests for real API/database behavior. |
| API docs | `internal/apidoc`, `docs/swagger` | OpenAPI catalog, generated Swagger JSON, and route coverage tests. |

## Main Business Modules

| Module | What it owns | Main endpoints |
|--------|--------------|----------------|
| Auth | Login, registration, refresh, tenant-token selection, logout, password reset. | `/api/v1/auth/*` |
| Tenants | Platform tenant creation, tenant profile update, branch creation. | `/api/v1/platform/tenants`, `/api/v1/admin/tenant` |
| Academic | Years, classes, sections, students, guardians, imports. | `/api/v1/admin/academic-years`, `/classes`, `/sections`, `/students`, `/guardians`, `/imports` |
| Billing | Fee heads, structures, assignments, invoices, student ledger, parent dues. | `/api/v1/admin/fee-*`, `/invoices`, `/students/:id/ledger`, `/parent/children/:id/dues` |
| Payments | Parent orders, verification, Razorpay webhooks, offline payments, receipts, events. | `/api/v1/parent/payments/*`, `/api/v1/webhooks/razorpay`, `/api/v1/admin/payments`, `/receipts` |
| Operations | Reminder templates/rules, reminder sends, logs, dashboard, reports, exports. | `/api/v1/admin/reminder-*`, `/dashboard`, `/reports/*`, `/exports` |

## How To Change A Feature

1. Start from the route in `internal/router/router.go`.
2. Read the handler method in `internal/handler`.
3. Follow the service method in `internal/service`.
4. Check repository SQL in `internal/repository/postgres`.
5. Add or update DTOs in `internal/dto`.
6. Add migrations when the data shape changes.
7. Update `internal/apidoc/catalog.go` for new or changed API routes.
8. Run `make swagger` and `make test`.
