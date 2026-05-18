---
name: code-quality
description: Use when reviewing a Go HTTP API codebase for readability, code quality, and idiomatic Go — specifically when systematic multi-dimensional coverage is needed across naming, idioms, complexity, consistency, and test quality
user-invocable: true
---

# Reviewing Go API: Readability & Code Quality Multi-Agent Review

## Overview

Dispatches 5 parallel specialized reviewer agents, each locked to one quality dimension of readability and code quality. Prevents the natural failure mode of a single sequential pass that mixes dimensions, builds no systematic checklist, and produces no severity-tagged findings.

**Counter to the "sequential builds context" argument:** Each agent builds context within their own lane — a naming auditor reads ALL modules through the naming lens and spots cross-module inconsistencies better than a mixed-pass reviewer who lost focus. A consistency auditor is explicitly tasked with cross-module patterns. The synthesis step catches anything that requires cross-dimension reasoning.

## When to Use

- Pre-merge review of a module or feature
- Codebase audit before onboarding new developers
- After a refactor — verifying consistency was maintained
- Any time "this code works but feels off" without knowing why

**Do NOT use for:** bug hunting (use `code-review`), single-file patches

## The 5 Reviewer Roles

### 1. Go Idioms Auditor

- Error wrapping: `fmt.Errorf("context: %w", err)` on every return, not bare `err`
- No naked returns or named return values (causes confusion at call sites)
- No `init()` functions — use explicit initialization in `main.go`
- No `panic()` in request path — only in `main.go` for fatal startup failures
- Interfaces defined at consumer, not producer — no "UserService" interface in the user package
- Interface size: single-method or narrow interfaces, not "do everything" god interfaces
- Receiver names: short and consistent per type (e.g., `s *Service`, not `this`, `self`, or varying names)
- Goroutine ownership: goroutines started with clear termination path
- `defer` used correctly — not in loops, not to defer work that should fail fast
- No `errors.New` inside functions — sentinel errors at package level
- `context.Background()` only in `main.go`, never mid-request

### 2. Naming & Clarity Auditor

- Exported types self-documenting without package prefix (e.g., `user.Service` not `user.UserService`)
- Variables named for meaning, not type (`count` not `n`, `userID` not `id` when ambiguous)
- Go acronym casing: `userID` not `userId`, `UUID` not `Uuid`, `httpClient` not `httpClient` if exported
- Functions named for what they DO (`EnrollStudent`, not `HandleEnrollment`)
- Boolean variables named as predicates (`isPublished`, `allowAnonymous`, not `published`, `anonymous`)
- No single-letter variables except loop indices and well-known short forms (`ctx`, `tx`, `r`, `w`)
- Comments only where WHY is non-obvious — no "// GetUser gets a user" comments
- Struct field names consistent with JSON tags and CLAUDE.md API field names (`snake_case` in JSON = meaningful Go name)
- No abbreviations unless universally understood (`req`/`resp` OK, `mgmt` not OK)
- Package names: lowercase, single-word, describe what they contain (not `util`, `common`, `helpers`)

### 3. Structure & Complexity Auditor

- Handler functions: validate → call service → write response (no business logic in handler)
- Service functions: business rules only, no HTTP types (`http.Request`, `http.ResponseWriter`)
- Repository functions: one DB operation per function, no business logic
- Function length: >40 lines = smell, >80 lines = refactor required
- Nesting depth: >3 levels of indentation = smell (use early returns)
- Single responsibility: one reason to change per function
- No "utility" functions doing 3 unrelated things
- Dependency injection via constructor, not method parameters
- State machines (`ValidateBatchTransition`, `ValidateFormTransition`) are pure functions — no side effects
- Error handling inline, not delegated to caller (no returned `error` ignored at call site)
- No duplicate code across modules — shared patterns extracted (pagination parsing, UUID parsing)

### 4. Consistency Auditor

Reads ALL modules and flags where patterns diverge from the established norm.

- Pagination: every list handler parses `page`/`per_page` the same way
- Error responses: every handler uses the same error envelope `{"error":{"code":"...","message":"..."}}`
- Logging: `slog.ErrorContext(ctx, "msg", "error", err, "field", val)` pattern identical across modules
- PATCH handlers: all check for zero-value vs. intentionally-set fields the same way
- Auth middleware application: every protected route uses the same middleware chain
- Handler function signature: `func (h *Handler) MethodResource(w http.ResponseWriter, r *http.Request)`
- Service constructor: `func New[Name]Service(pool *pgxpool.Pool, q *db.Queries) *[Name]Service`
- Repository: all use `*db.Queries`, not raw `*pgxpool.Pool` for queries
- UUID parsing from path params: same helper across all handlers (no inline `uuid.Parse` variations)
- Soft delete: all modules use the correct strategy per CLAUDE.md (status vs. deleted_at vs. dropped)

### 5. Test Quality Auditor

- Test naming: `Test[FunctionName]_[scenario]` (e.g., `TestEnrollStudent_CapacityFull`)
- Table-driven tests for multiple input/output variations (not duplicate test functions)
- No logic in tests: test data setup is plain values, not computed
- Assertions use `testify/assert` or `testify/require` (not manual `if err != nil { t.Fatal }`)
- `require` for preconditions (test can't continue if fails), `assert` for all-cases assertions
- No production code changed to make tests pass (no exported fields/methods added only for tests)
- Test helpers in `*_test.go` files, not in production packages
- Integration tests hit real DB (no mocking pgx) — see CLAUDE.md preference
- Test isolation: each test cleans up its own DB state (no test order dependency)
- Error path tests exist, not just happy path
- No `time.Sleep` in tests — use condition-based waiting

## Dispatch Instructions

**REQUIRED:** Use `superpowers:dispatching-parallel-agents`. Send all 5 agents simultaneously. Do NOT run sequentially — each agent builds context within their own dimension, which is more thorough than a mixed sequential pass.

Prompt template per agent:

```
You are a [ROLE, e.g. "Go Idioms Auditor"] reviewing a Go 1.25 HTTP API for readability and code quality.

Stack: chi router, pgx/v5 + pgxpool, sqlc, JWT RS256, slog, Sentry. Modular monolith: internal/module/[name]/ has handler.go, service.go, repository.go. No ORM, no DI framework, explicit dependency injection.

Your EXCLUSIVE focus: [paste the relevant checklist section from above]

Directories to review:
- internal/module/ — all modules
- internal/middleware/
- internal/worker/
- internal/config/
- cmd/server/main.go

For each finding, use exactly:
SEVERITY: HIGH | MEDIUM | LOW
FILE: path/to/file.go:line
ISSUE: what is wrong and why it harms readability/quality
FIX: specific, actionable fix

No CRITICAL severity for readability issues. Skip if it's purely a matter of taste.
Only report findings that would confuse a new developer or cause bugs from misreading.
```

## Output Synthesis

After all 5 agents complete:

```
## Go Idioms [N findings]
HIGH: FILE path:line — issue
MEDIUM: ...

## Naming & Clarity [N findings]
...

## Structure & Complexity [N findings]
...

## Consistency [N findings]
...

## Test Quality [N findings]
...

## Top Readability Improvements (cross-dimension)
1. [SEVERITY] Domain — specific fix
```

## Common Mistakes

| Mistake                                                 | Fix                                                                                            |
| ------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| Running agents sequentially because "context matters"   | Each agent builds its own dimension's context; consistency agent handles cross-module patterns |
| Raising bugs or security issues                         | That's `code-review` — readability only                                                        |
| Reporting style preferences ("I'd name it differently") | Only report what harms comprehension or causes misreading                                      |
| Consistency agent reports within-module issues          | Consistency agent compares ACROSS modules only                                                 |
| Skipping test quality                                   | Test code is production code — same quality bar                                                |
| Using CRITICAL severity                                 | Readability max severity is HIGH                                                               |
