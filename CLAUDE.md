# eduwallet

## Architecture

Layered architecture: Handler -> Service -> Repository -> Database

- `cmd/api/main.go` - Entry point with DI wiring
- `internal/config/` - Environment-based configuration
- `internal/database/` - Postgres pool, Redis client, DBTX interface
- `internal/apperror/` - Structured error types (AppError with machine-readable codes)
- `internal/dto/` - Request/response data transfer objects
- `internal/handler/` - HTTP handlers (Gin), bind/validate/respond
- `internal/middleware/` - Auth, CORS, logger, rate limit, recovery, request ID, role guard, security headers
- `internal/model/` - Domain models
- `internal/repository/` - Data access interfaces + Postgres implementations (DBTX-based)
- `internal/router/` - Route definitions with middleware chains
- `internal/service/` - Business logic
- `pkg/` - Shared packages (email, hasher, jwt, logger)

## Conventions

- All repositories accept `database.DBTX` interface (works with pool or transaction)
- Services return `*apperror.AppError` for domain errors (machine-readable codes)
- Handlers call `HandleError(c, err)` for all errors (single mapper)
- DTOs are separate from models (request != response != database)
- UUIDs for all primary keys
- Soft delete via `deleted_at` timestamp
- All list endpoints support pagination (page, page_size, sort_by, sort_dir)

## Adding a New Entity

1. Create model: `internal/model/product.go`
2. Create DTOs: `internal/dto/product.go` (CreateRequest, UpdateRequest, Response)
3. Add error codes: `internal/apperror/apperror.go` (ErrProductNotFound, etc.)
4. Add repo interface: `internal/repository/interfaces.go`
5. Implement repo: `internal/repository/postgres/product.go` (accepts DBTX)
6. Add service interface: `internal/service/interfaces.go`
7. Implement service: `internal/service/product.go`
8. Create handler: `internal/handler/product.go`
9. Add routes: `internal/router/router.go` (find "ADD YOUR ROUTES HERE")
10. Wire DI: `cmd/api/main.go` (find "ADD YOUR" markers)
11. Create migration: `make migrate-create name=create_products`

## Testing

- `make test` - Unit tests with mocks (<3 seconds)
- `make test-e2e` - E2E tests with testcontainers (real Postgres + Redis)
- Unit tests use table-driven patterns (see tests/unit/)
- Mocks are in tests/mocks/ (hand-written, testify/mock)

## Common Commands

- `make dev` - Hot reload development
- `make build` - Build binary
- `make migrate-up` - Run migrations
- `make migrate-create name=X` - New migration
- `make docker-up` - Start Postgres + Redis
- `make swagger` - Generate API docs
- `make lint` - Run linter
- `make test` - Fast unit tests
- `make test-e2e` - E2E tests
