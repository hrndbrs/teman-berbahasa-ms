# CLAUDE.md — Teman Berbahasa Microservice

## Project Overview

Go backend for **Teman Berbahasa Tutor Place Management System** — an internal web-based management system for a tutoring center. Staff/teacher-facing; no student-facing portal.

**Module:** `github.com/hrndbrs/teman-berbahasa-ms`  
**Go version:** 1.25

## Architecture

**Modular monolith.** Single binary, single PostgreSQL instance.

```
cmd/
└── server/
    └── main.go              # entry point: config, DI, server start
internal/
├── config/                  # env loading + validation
├── db/                      # sqlc generated code + migrations
│   ├── migrations/
│   └── query/               # sqlc generated *.go files
├── middleware/              # auth, logging, recovery, CORS
├── patch/                   # Patchable[T] generic type for nullable PATCH fields
├── worker/                  # goroutine pool, job types, dispatcher
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
        └── response/        # sub-module
```

Each module has `handler.go`, `service.go`, `repository.go` and exposes a `Register(r Router, deps Deps)` function — no global state.

## Key Libraries

| Concern           | Library                                    |
| ----------------- | ------------------------------------------ |
| HTTP router       | `github.com/go-chi/chi/v5`                 |
| PostgreSQL driver | `github.com/jackc/pgx/v5` + `pgxpool`      |
| Query generation  | `github.com/sqlc-dev/sqlc`                 |
| Migrations        | `github.com/golang-migrate/migrate/v4`     |
| JWT               | `github.com/golang-jwt/jwt/v5` (RS256)     |
| Password hashing  | `golang.org/x/crypto/bcrypt` (cost 12)     |
| UUID              | `github.com/google/uuid` (UUID v7 for PKs) |
| Validation        | `github.com/go-playground/validator/v10`   |
| Logging           | `log/slog` (stdlib, structured JSON)       |
| Error tracking    | `github.com/getsentry/sentry-go`           |

No ORM. Raw SQL via sqlc. No Redis — async jobs via in-process goroutine worker pool.

## Modules

| Module              | Endpoints prefix               | Purpose                                        |
| ------------------- | ------------------------------ | ---------------------------------------------- |
| Auth                | `/auth/*`                      | Login, logout, token refresh, password reset   |
| User Management     | `/users`                       | CRUD for admin/teacher/staff accounts          |
| Student Management  | `/students`                    | CRUD for student records                       |
| Course Management   | `/courses`                     | Course catalog CRUD + archive                  |
| Batch Management    | `/batches`                     | Batches scoped to a course, status transitions |
| Enrollment          | `/enrollments`                 | Enroll students, payment tracking, final grade |
| Schedule Management | `/batches/:id/schedules`, etc. | Recurring timetable slots + one-off overrides  |
| Event Management    | `/events`                      | Institution-wide calendar events               |
| Form Management     | `/forms`, `/form-questions`    | Dynamic form builder with full lifecycle       |
| Form Response       | `/forms/:id/responses`         | Accept submissions, bulk insert answers        |

## API Conventions

- All IDs: UUID v7 string — `"019687a2-1234-7abc-8def-000000000001"`
- All datetimes: ISO 8601 UTC — `"2025-05-18T07:00:00Z"`
- All dates: `"2025-05-18"`, all times: `"09:00:00"` (24h, WIB/UTC+7). Time inputs accept both `"HH:MM"` and `"HH:MM:SS"`; responses always return `"HH:MM:SS"`.
- Field names: `snake_case` throughout
- `PATCH` is partial — only sent fields are updated. Nullable fields support explicit `null` to clear: send `"field": null` to set to NULL, omit the key to leave unchanged. Implemented via `internal/patch.Patchable[T]` with custom `UnmarshalJSON`.
- Enums: lowercase strings (never integers)
- `price` returned as string to avoid float precision loss
- Nested objects in responses — never bare IDs in response bodies
- `created_by_user_id` always server-set from JWT; never accepted from client
- Passwords never appear in any response; `password_hash` never leaves DB
- Pagination: `?page=1&per_page=20` (max 100), response includes `pagination` object

### Error Envelope

```json
{ "error": { "code": "VALIDATION_ERROR", "message": "...", "fields": {} } }
```

Standard codes: `BAD_REQUEST` (400), `UNAUTHORIZED` (401), `FORBIDDEN` (403), `NOT_FOUND` (404), `CONFLICT` (409), `GONE` (410), `VALIDATION_ERROR` (422), `RATE_LIMITED` (429), `INTERNAL_ERROR` (500).

`POST /auth/login` additionally returns `401 ACCOUNT_LOCKED` when the account is locked after 10 failed attempts.

## Database

- PostgreSQL, single instance. All PKs are UUID v7.
- `pgx/v5` driver with `pgxpool` — pool config: `MaxConns=20`, `MinConns=2`, `MaxConnLifetime=1h`, `MaxConnIdleTime=10m`, `HealthCheckPeriod=30s`.
- All queries via sqlc — no string concatenation in queries.
- All `TIMESTAMPTZ` columns store UTC.
- `day_of_week` interpreted in WIB (UTC+7).

### Soft Deletes & Audit

| Table                | Strategy                                        |
| -------------------- | ----------------------------------------------- |
| `users`              | `status = inactive` — never hard delete         |
| `students`           | `status = inactive` — never hard delete         |
| `forms`              | `deleted_at TIMESTAMPTZ` + `status = deleted`   |
| `enrollments`        | `status = dropped` — preserved for audit        |
| `courses`            | `status = archived`                             |
| `refresh_tokens`     | Hard delete on logout; cron purges expired rows |
| `schedule_overrides` | Hard delete allowed                             |

### Critical Unique Constraints

| Constraint                           | On table             |
| ------------------------------------ | -------------------- |
| `email`                              | `users`, `students`  |
| `(course_id, batch_code)`            | `batches`            |
| `(student_id, batch_id)`             | `enrollments`        |
| `(schedule_id, original_date)`       | `schedule_overrides` |
| `(form_id, respondent_id)` (partial) | `form_responses`     |
| `course_code`                        | `courses`            |

Catch `pgerrcode.UniqueViolation` → return 409.

## Business Rules

### Auth

- bcrypt cost 12. JWT signed with RS256 (`jwt.WithValidMethods([]string{"RS256"})`). Access token TTL: 15 min. Refresh token TTL: 7 days.
- Refresh token stored as `SHA-256(raw_token)` — raw token never persists in DB.
- Refresh rotation wrapped in a single transaction with `SELECT ... FOR UPDATE` on the token row — prevents duplicate session issuance under concurrent requests.
- Reusing a rotated token → 401. Concurrent requests with the same token: first wins, second gets 401.
- Account lock after 10 failed login attempts; reset on success.
- `Login`: runs dummy bcrypt when email not found to prevent timing-oracle enumeration.
- `ForgotPassword`: silently returns nil for inactive or unknown accounts (anti-enumeration).
- `ResetPassword`: checks `user.Status = active` before accepting the token; wraps password update + token deletion + session revocation in one transaction.

### Batches

- `instructor_user_id` must reference a `users.role = teacher` — validate in service before insert.
- Status transitions: `upcoming → ongoing → completed` only. No reversals.
- Cannot delete batch with active enrollments — enforced atomically via `DELETE ... WHERE NOT EXISTS (active enrollments) RETURNING id`. No check-then-act race.
- **No `start_date`/`end_date` on batches.** Date range is derived from schedules (`MIN`/`MAX` of schedule slots). Storing dates on the batch would require double-updating whenever a schedule is rescheduled (e.g., due to a holiday). `academic_year` (e.g., `"2026"`) is kept as a standalone administrative label.
- When schedule module is built, add `first_class_date` and `last_class_date` to `batches_with_stats` view via `MIN`/`MAX` of schedule effective dates.

### Enrollments

- `course_id` server-derived from `batch.course_id` — never trust client-provided value.
- Cannot enroll into a `completed` batch.
- Capacity check: `COUNT(enrollments WHERE status != 'dropped') < course.max_capacity`.
- Race condition: use `pg_advisory_xact_lock(batch_id)` inside serializable transaction. Catch `pgerrcode.SerializationFailure` (40001) and retry once.

### Schedules / Overrides

- Effective instructor resolution (3-level fallback): override → slot → batch default.
- `original_date` must fall within `schedule.effective_from` / `effective_until`.
- `cancellation` override: `new_date`, `new_start_time`, `new_end_time` must be nil.
- `reschedule` override: `new_date` required.

### Forms

- Cannot publish with zero questions.
- Questions immutable once form is `published` — return 409 on edit attempt.
- `POST /forms/:id/publish` and `POST /forms/:id/close` are idempotent — return 200 if already in target state.
- `allow_anonymous = false` → respondent name + email required on submission.
- Partial unique index on `(form_id, respondent_id)` prevents duplicate identified submissions.

### Form Submission

- Validate all `is_required = true` questions answered.
- Validate all `answer.question_id` belong to the form before starting transaction.
- Single transaction: upsert respondent → insert response → bulk insert answers via `unnest`.
- After commit: dispatch `FormResponseSubmitted` event to worker channel (non-blocking).

### Courses

- Cannot archive a course with `ongoing` batches — enforced atomically via `UPDATE courses SET status='archived' WHERE ... AND NOT EXISTS (ongoing batches) RETURNING id`. No check-then-act race.

## State Machines

```
Batch:      upcoming → ongoing → completed  (no reversal)
Enrollment: enrolled → dropped | completed
Payment:    pending → partial | paid → paid  (no reversal)
Form:       draft → published → closed → deleted (soft)
            draft → deleted (soft)
            published → deleted (soft)
```

Implement as explicit `ValidateBatchTransition` / `ValidateFormTransition` functions.

## Roles & Permissions

| Action                    | Admin |        Teacher        | Staff |
| ------------------------- | :---: | :-------------------: | :---: |
| Manage users              |  ✅   |          ❌           |  ❌   |
| Manage students           |  ✅   |          ❌           |  ✅   |
| View students             |  ✅   |          ✅           |  ✅   |
| Manage courses            |  ✅   |          ❌           |  ❌   |
| Manage batches            |  ✅   |          ❌           |  ✅   |
| Manage enrollments        |  ✅   |          ❌           |  ✅   |
| Manage schedules          |  ✅   |          ❌           |  ✅   |
| Create schedule overrides |  ✅   | ✅ (own batches only) |  ✅   |
| Create/edit/delete forms  |  ✅   |          ❌           |  ✅   |
| View form responses       |  ✅   |          ❌           |  ✅   |
| Submit form responses     |  ✅   |          ✅           |  ✅   |

Role enforcement in middleware, not handler logic.  
`GET /users/:id`: own record only unless admin.  
Teachers can only create schedule overrides for batches where `batch.instructor_user_id = ctx_user_id`.

## Auth Middleware Pattern

```go
func AuthMiddleware(next http.Handler) http.Handler
func RequireRole(roles ...string) func(http.Handler) http.Handler
```

## Dependency Injection

No DI framework. Pass dependencies explicitly. Wire everything in `main.go`.

```go
type FormService struct {
    db      *pgxpool.Pool
    queries *db.Queries   // sqlc generated
    worker  *worker.Worker
}
func NewFormService(pool *pgxpool.Pool, q *db.Queries, w *worker.Worker) *FormService
```

## Worker Pool

Channel-based goroutine pool — no Redis. `Dispatch` is non-blocking; drops job if channel full (acceptable for notification fanout in v1).

- `Job` interface requires `Type() string`, `Execute(ctx)`, and `LogFields() []any` — `LogFields` provides entity IDs for structured logging on drop.
- Dropped jobs logged at WARN with entity fields + running `dropped_total` counter (accessible from `Worker.DroppedTotal()`).
- Worker goroutine panics: logged + Sentry-captured, then restarted with exponential backoff (1s × restart count, capped at 30s).
- `Worker.Drain(timeout)` waits for in-flight jobs to finish (uses `sync.WaitGroup`). Call after `http.Server.Shutdown` and before `pool.Close`.
- Shutdown order in `main.go`: `srv.Shutdown → w.Drain(10s) → sentry.Flush(2s) → pool.Close`.
- **Horizontal scaling note:** the in-process pool is not shared across instances. Before adding a second replica, replace with an external queue (e.g. PostgreSQL `LISTEN/NOTIFY` or PgMQ).

## Security

- All endpoints require JWT except `POST /forms/:id/responses` (when `allow_anonymous = true`).
- HTTPS via Caddy reverse proxy (auto TLS).
- Secrets via env vars only — never hardcoded.
- All DB queries parameterized via sqlc.
- Strip leading/trailing whitespace on all string fields at request parsing.
- CORS: restrict `Access-Control-Allow-Origin` to known frontend origins.
- All request bodies limited to 1 MB via `http.MaxBytesReader` in every `decode()` helper.
- Bearer token strings capped at 4096 bytes before JWT parsing.
- JWT algorithm pinned to RS256 via `jwt.WithValidMethods([]string{"RS256"})` — rejects `none`, HS256, and other RSA variants.
- Panics in HTTP handlers and worker goroutines are captured to Sentry via `sentry.CurrentHub().RecoverWithContext`.

## Non-Functional

- Target: <100 concurrent users, <10k students.
- p95 response time < 300ms. Form submission < 500ms.
- Structured JSON logs via `log/slog`: fields `method`, `path`, `user_id`, `status`, `duration_ms`, `error`.
- `GET /health` → `{"status":"ok","db":"ok"}`.
- Timezone: all TIMESTAMPTZ in UTC; `day_of_week` in WIB (UTC+7).
- Use `RETURNING` clauses (or CTEs joining the view) to avoid extra SELECT after INSERT/UPDATE.
- Graceful shutdown: `signal.NotifyContext` + `http.Server.Shutdown(ctx)` with 30s timeout, followed by worker drain and Sentry flush before pool close.
- `GET /students/:id` enrollment history capped at 50 most recent (`LIMIT 50` in `GetStudentEnrollments`).

## Deployment

- Single VPS (2 vCPU / 4 GB RAM).
- Docker Compose: `api` + `postgres` + `caddy`.
- Build: `CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server`
- CI/CD: GitHub Actions → build → docker build → push → SSH `docker compose up -d`
