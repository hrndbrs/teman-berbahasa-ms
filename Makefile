BINARY     := server
CMD        := ./cmd/server
MIGRATIONS := internal/db/migrations
SQLC_DIR   := internal/db

DATABASE_URL ?= $(shell grep -E '^DATABASE_URL=' .env 2>/dev/null | cut -d= -f2-)

.PHONY: run build clean test test-race vet tidy \
        migrate-up migrate-down migrate-create \
        sqlc keys \
        docker-up docker-down docker-logs

# ── Dev ──────────────────────────────────────────────────────────────────────

run:
	go run $(CMD)

build:
	CGO_ENABLED=0 GOOS=linux go build -o $(BINARY) $(CMD)

clean:
	rm -f $(BINARY)

# ── Test & Quality ───────────────────────────────────────────────────────────

test:
	go test ./...

test-race:
	go test -race ./...

test-integration:
	go test -race ./... -count=1

vet:
	go vet ./...

tidy:
	go mod tidy

# ── Database ─────────────────────────────────────────────────────────────────

migrate-up:
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" down

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir $(MIGRATIONS) -seq $$name

# ── Code Generation ──────────────────────────────────────────────────────────

sqlc:
	cd $(SQLC_DIR) && sqlc generate

# ── Keys ─────────────────────────────────────────────────────────────────────

keys:
	mkdir -p keys
	openssl genrsa -out keys/private.pem 2048
	openssl rsa -in keys/private.pem -pubout -out keys/public.pem

# ── Docker ───────────────────────────────────────────────────────────────────

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f api
