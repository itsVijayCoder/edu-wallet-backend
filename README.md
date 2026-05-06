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
| POST   | `/api/v1/auth/logout`             | Logout (requires auth)     |
| POST   | `/api/v1/auth/forgot-password`    | Request password reset     |
| POST   | `/api/v1/auth/reset-password`     | Reset password with token  |

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
| `RESEND_API_KEY`     | -                | Resend API key (optional, for emails) |

## Testing

```bash
make test        # Unit tests with race detector (<3 seconds)
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
