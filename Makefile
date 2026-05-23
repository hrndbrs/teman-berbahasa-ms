BINARY      := server
CMD         := ./cmd/server
MIGRATIONS  := internal/db/migrations
SQLC_DIR    := internal/db
MIGRATE     := go run ./cmd/migrate
MIGRATE_CLI := go run github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1

DATABASE_URL ?= $(shell grep -E '^DATABASE_URL=' .env 2>/dev/null | cut -d= -f2-)

.PHONY: run dev build build-local clean \
        test test-race test-integration coverage \
        fmt fmt-check vet lint tidy check ci \
        migrate-up migrate-down migrate-down-one migrate-up-one \
        migrate-version migrate-status migrate-force migrate-create db-reset \
        seed sqlc keys \
        docker-up docker-down docker-build docker-rebuild \
        docker-restart docker-logs docker-shell \
        install-tools help

# ── Help ─────────────────────────────────────────────────────────────────────

help: ## Show all available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'

# ── Dev ──────────────────────────────────────────────────────────────────────

run: ## Run server without live reload
	go run $(CMD)

dev: ## Run with live reload (requires: make install-tools)
	air

build: ## Build production binary (Linux, stripped)
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o $(BINARY) $(CMD)

build-local: ## Build for current OS/arch (dev use)
	go build -o $(BINARY) $(CMD)

clean: ## Remove built binary and coverage artifacts
	rm -f $(BINARY) coverage.out coverage.html

# ── Test & Quality ───────────────────────────────────────────────────────────

test: ## Run all tests
	go test ./...

test-race: ## Run tests with race detector
	go test -race ./...

test-integration: ## Run all tests, bypass cache
	go test -race -count=1 ./...

coverage: ## Generate and open HTML coverage report
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	open coverage.html

fmt: ## Format all Go source files
	gofmt -w .

fmt-check: ## Fail if any files are unformatted
	@test -z "$$(gofmt -l .)" || (echo "Unformatted files:"; gofmt -l .; exit 1)

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint (requires: make install-tools)
	golangci-lint run ./...

tidy: ## Tidy go.mod and go.sum
	go mod tidy

check: vet lint fmt-check ## Run vet + lint + fmt-check

ci: tidy check test-race ## Full CI pipeline

# ── Database ─────────────────────────────────────────────────────────────────

migrate-up: ## Apply all pending migrations
	$(MIGRATE) -path $(MIGRATIONS) -database "$(DATABASE_URL)" up

migrate-down: ## Roll back ALL migrations (destructive)
	$(MIGRATE) -path $(MIGRATIONS) -database "$(DATABASE_URL)" down

migrate-up-one: ## Apply one migration step
	$(MIGRATE_CLI) -path $(MIGRATIONS) -database "$(DATABASE_URL)" up 1

migrate-down-one: ## Roll back one migration step
	$(MIGRATE_CLI) -path $(MIGRATIONS) -database "$(DATABASE_URL)" down 1

migrate-version: ## Show current migration version and dirty flag
	$(MIGRATE) -path $(MIGRATIONS) -database "$(DATABASE_URL)" version

migrate-status: ## List all migrations with applied [x]/[ ] indicators
	$(MIGRATE) -path $(MIGRATIONS) -database "$(DATABASE_URL)" status

migrate-force: ## Force migration to version N (fix dirty state): make migrate-force V=3
	$(MIGRATE_CLI) -path $(MIGRATIONS) -database "$(DATABASE_URL)" force $(V)

migrate-create: ## Create a new migration file (prompts for name)
	@read -p "Migration name: " name; \
	$(MIGRATE_CLI) create -ext sql -dir $(MIGRATIONS) -seq $$name

db-reset: migrate-down migrate-up ## Full DB reset: roll back all then reapply

seed: ## Run database seeder
	go run ./cmd/seed

# ── Code Generation ──────────────────────────────────────────────────────────

sqlc: ## Regenerate sqlc query code
	cd $(SQLC_DIR) && sqlc generate

# ── Keys ─────────────────────────────────────────────────────────────────────

keys: ## Regenerate RSA-2048 key pair for JWT signing
	openssl genrsa -out private.pem 2048
	openssl rsa -in private.pem -pubout -out public.pem

# ── Docker ───────────────────────────────────────────────────────────────────

docker-up: ## Start all services detached
	docker compose up -d

docker-down: ## Stop and remove containers
	docker compose down

docker-build: ## Build api image (no cache)
	docker compose build --no-cache api

docker-rebuild: ## Rebuild api image and restart container
	docker compose up -d --build api

docker-restart: ## Restart api container
	docker compose restart api

docker-logs: ## Tail api logs
	docker compose logs -f api

docker-shell: ## Open shell in running api container
	docker compose exec api sh

# ── Tools ────────────────────────────────────────────────────────────────────

install-tools: ## Install dev tools: air, golangci-lint, sqlc
	go install github.com/air-verse/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
