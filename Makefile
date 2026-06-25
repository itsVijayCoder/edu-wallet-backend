include .env
export

BINARY     := bin/eduwallet
MIGRATE_BIN := bin/eduwallet-migrate
DB_DSN     := postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)
MIGRATE    := migrate -database "$(DB_DSN)" -path migrations

## ─── Development ─────────────────────────────────────────────

.PHONY: dev
dev: ## Run with hot-reload (requires air)
	air

.PHONY: build
build: ## Build production binary
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BINARY) ./cmd/api

.PHONY: run
run: build ## Build and run
	./$(BINARY)

.PHONY: clean
clean: ## Remove build artefacts
	rm -rf bin/ tmp/

## ─── Testing ─────────────────────────────────────────────────

.PHONY: test
test: ## Run unit tests (short mode, race detector)
	go test -short -race -count=1 ./...

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests
	go test -v -timeout=15m ./tests/e2e/...

.PHONY: test-all
test-all: ## Run all tests (unit + e2e)
	go test -race -count=1 -timeout=15m ./...

.PHONY: coverage
coverage: ## Run tests with coverage report
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## ─── Code Quality ────────────────────────────────────────────

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format code
	gofmt -w .
	goimports -w .

.PHONY: vet
vet: ## Run go vet
	go vet ./...

## ─── Migrations ──────────────────────────────────────────────
##
## Migration targets use the bundled cmd/migrate binary, so the external
## `migrate` CLI is no longer required. The DSN is resolved from .env via
## config.Load (DATABASE_URL takes precedence over the DB_* vars).

.PHONY: migrate
migrate: ## Build the migrate binary
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(MIGRATE_BIN) ./cmd/migrate

.PHONY: migrate-up
migrate-up: migrate ## Apply all pending migrations
	$(MIGRATE_BIN) up

.PHONY: migrate-down
migrate-down: migrate ## Roll back last migration (usage: make migrate-down n=3)
	$(MIGRATE_BIN) down $(n)

.PHONY: migrate-create
migrate-create: migrate ## Create new migration (usage: make migrate-create name=create_foo_table)
	$(MIGRATE_BIN) create $(name)

.PHONY: migrate-version
migrate-version: migrate ## Print current migration version
	$(MIGRATE_BIN) version

.PHONY: migrate-force
migrate-force: migrate ## Force migration version (usage: make migrate-force version=1)
	$(MIGRATE_BIN) force $(version)

.PHONY: migrate-goto
migrate-goto: migrate ## Migrate to version (usage: make migrate-goto version=5)
	$(MIGRATE_BIN) goto $(version)

## ─── Docker ──────────────────────────────────────────────────

.PHONY: docker-up
docker-up: ## Start all services
	docker compose up -d --build

.PHONY: docker-down
docker-down: ## Stop all services
	docker compose down

.PHONY: docker-logs
docker-logs: ## Tail service logs
	docker compose logs -f

## ─── CI ──────────────────────────────────────────────────────

.PHONY: ci-test
ci-test: ## CI test pipeline
	go test -race -count=1 -coverprofile=coverage.out ./...

.PHONY: ci-lint
ci-lint: ## CI lint pipeline
	golangci-lint run --out-format=github-actions ./...

## ─── Swagger ─────────────────────────────────────────────────

.PHONY: swagger
swagger: ## Generate Swagger docs
	go run ./cmd/apidocs

.PHONY: swagger-fmt
swagger-fmt: ## Format Swagger comments
	$(MAKE) swagger

## ─── Deployment ──────────────────────────────────────────────

.PHONY: upgrade
upgrade: ## Pull latest, migrate, rebuild and restart
	git pull
	$(MIGRATE) up
	$(MAKE) build
	@echo "Restarting eduwallet..."
	-pkill -f $(BINARY) || true
	nohup ./$(BINARY) > /tmp/eduwallet.log 2>&1 &
	@echo "eduwallet restarted. Logs: /tmp/eduwallet.log"

.PHONY: stop
stop: ## Stop the running server
	-pkill -f $(BINARY) || true
	@echo "eduwallet stopped"

## ─── Help ────────────────────────────────────────────────────

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
