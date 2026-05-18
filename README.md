# Teman Berbahasa — Tutor Place Management System

Backend for the internal management system of Teman Berbahasa, a tutoring center. Used by admin, staff, and teachers to run day-to-day operations: registering students, managing courses and class batches, tracking enrollments and payments, scheduling sessions, and collecting feedback through forms.

Not student-facing. This is the tool the people running the center use.

---

## What it manages

**Students** — registration, profiles, parent contacts, enrollment history across batches.

**Courses & Batches** — a course (e.g. "English Intermediate") can run as multiple batches in parallel or across academic years. Each batch has an assigned instructor, a schedule, and a capacity limit.

**Enrollments** — students enroll into batches. The system tracks payment status (`pending → partial → paid`) and final grade. Capacity is enforced with database-level advisory locks to prevent double-enrollment under concurrent requests.

**Schedules** — recurring weekly slots or one-time sessions per batch. Individual sessions can be overridden (rescheduled, cancelled, or reassigned to a different instructor) without touching the base schedule.

**Events** — institution-wide calendar events (exams, workshops, holidays) targeting all staff, students, or specific batches.

**Forms** — a dynamic form builder. Staff create forms, add questions, publish them, and collect responses. Published forms lock their questions. Anonymous submissions supported. Responses viewable and exportable as CSV.

---

## Architecture

Single Go binary backed by a single PostgreSQL instance — no microservices, no message broker, no cache layer.

```
cmd/server/main.go          — entry point, DI wiring, server start
internal/
├── config/                 — env loading and startup validation
├── db/
│   ├── migrations/         — SQL migration files (golang-migrate)
│   └── query/              — sqlc-generated type-safe query code
├── middleware/             — auth, logging, recovery, CORS
├── worker/                 — channel-based goroutine pool
└── module/
    ├── auth/
    ├── user/
    ├── student/
    ├── course/
    ├── batch/
    ├── enrollment/
    ├── schedule/
    ├── event/
    └── form/
        └── response/
```

Each module has `handler.go`, `service.go`, `repository.go` and wires itself via `Register(router, deps)`. No global state. All dependencies injected explicitly in `main.go`.

---

## Stack

| Concern | Library |
|---|---|
| HTTP router | `go-chi/chi/v5` |
| PostgreSQL | `jackc/pgx/v5` + `pgxpool` |
| Query generation | `sqlc-dev/sqlc` — no ORM, raw SQL |
| Migrations | `golang-migrate/migrate/v4` |
| JWT | `golang-jwt/jwt/v5` — RS256 |
| Password hashing | `golang.org/x/crypto/bcrypt` — cost 12 |
| Primary keys | `google/uuid` — UUID v7 (time-ordered) |
| Validation | `go-playground/validator/v10` |
| Logging | `log/slog` — structured JSON |
| Error tracking | `getsentry/sentry-go` |

---

## Auth

- RS256 JWT. Access token TTL: 15 min. Refresh token TTL: 7 days.
- Refresh tokens stored as `SHA-256(raw_token)` — the raw token never touches the database.
- Tokens rotate on every refresh. Reusing a rotated token returns 401 immediately.
- 10 failed login attempts locks the account. Resets on successful login.

Three roles: `admin`, `teacher`, `staff`. Role checks live in middleware — handlers never inspect roles directly.

---

## API conventions

- IDs: UUID v7 string — `"019687a2-1234-7abc-8def-000000000001"`
- Datetimes: ISO 8601 UTC — `"2025-05-18T07:00:00Z"`
- Field names: `snake_case` throughout
- `PATCH` is partial — only sent fields update
- Responses always nest full objects — never bare foreign-key IDs
- `price` returned as string (no float precision loss)

**Error shape:**

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "fields": { "email": "must be a valid email address" }
  }
}
```

**List endpoints** all paginate: `?page=1&per_page=20` (max 100), response includes a `pagination` object.

---

## Local setup

### Prerequisites

- Go 1.25+
- PostgreSQL 15+
- `sqlc` — [sqlc.dev](https://sqlc.dev)
- `migrate` CLI — [golang-migrate](https://github.com/golang-migrate/migrate)

### Environment variables

```env
DATABASE_URL=postgres://user:pass@localhost:5432/teman_berbahasa?sslmode=disable
JWT_PRIVATE_KEY_PATH=./keys/private.pem
JWT_PUBLIC_KEY_PATH=./keys/public.pem
SENTRY_DSN=
CORS_ALLOWED_ORIGINS=http://localhost:3000
```

Missing required vars cause immediate startup failure — nothing is silently defaulted.

### Run

```bash
# Generate RS256 key pair (first time only)
openssl genrsa -out keys/private.pem 2048
openssl rsa -in keys/private.pem -pubout -out keys/public.pem

# Apply migrations
migrate -path internal/db/migrations -database "$DATABASE_URL" up

# Start
go run ./cmd/server
```

### Build

```bash
CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server
```

### Regenerate queries (after editing `.sql` files)

```bash
sqlc generate
```

---

## Deployment

Single VPS (2 vCPU / 4 GB RAM). Docker Compose: `api` + `postgres` + `caddy` (auto TLS via Caddy).

```bash
docker compose up -d
```

CI/CD pipeline: GitHub Actions → build → Docker image → push to registry → SSH deploy with `docker compose up -d --no-deps --build api`.

Graceful shutdown on `SIGTERM`/`SIGINT`: 30-second drain window before hard stop.

```
GET /health  →  {"status":"ok","db":"ok"}
```

---

## Docs

- [`docs/data-contract.md`](docs/data-contract.md) — every endpoint's request and response shape
- [`docs/erd.md`](docs/erd.md) — entity-relationship diagram with rationale per column
- [`docs/functional-requirement-diagram.md`](docs/functional-requirement-diagram.md) — module flows, role matrix, state machines, edge case handling
