---
name: code-review
description: Use when reviewing a Go HTTP API codebase for security, architecture, performance, reliability, and scalability — specifically when systematic multi-dimensional coverage is needed and single-pass review risks blind spots or dimension mixing
user-invocable: true
---

# Reviewing Go API: Multi-Agent Codebase Review

## Overview

Dispatches 5 parallel specialized reviewer agents, each locked to one quality dimension. Produces severity-tagged, file:line findings per role, then synthesizes a cross-dimension priority list. Prevents the natural failure mode of sequential mixed reviews with no systematic checklist.

## When to Use

- Pre-release or pre-merge review of a Go HTTP service
- Architecture audit across module boundaries
- Catching security/reliability issues before load testing
- Any review where single-agent scope drift is a risk

**Do NOT use for:** single-file patches, quick bug checks, style review

## The 5 Reviewer Roles

### 1. Security Engineer

- JWT: algorithm enforcement (reject none/HS256 if RS256 expected), expiry check uses `<=` not `<`
- Refresh tokens: SHA-256 hash stored (not raw), rotated on every use, reuse → 401
- bcrypt: cost ≥ 12, constant-time comparison
- Account lockout: failed_attempts incremented on failure, reset on success, threshold enforced
- Password reset: token generation uses `crypto/rand` (≥32 bytes), storage as hash, table exists in schema
- Authorization: role checks in middleware not handlers; teacher scope (own batches only) enforced
- CORS: restricted to known origins, not wildcard
- Secrets: env vars only, startup validation fails fast if missing; never hardcoded, never logged
- Password/hash fields: never in API responses, never in logs
- Input: whitespace stripped on all string fields at request boundary
- Anonymous endpoints: only explicitly intended ones; rate limiting present
- SQL: all queries parameterized (sqlc), no string concatenation anywhere

### 2. Backend Architect

- Layer discipline: handler → service → repository, no layer skipping
- Every module has handler.go, service.go, repository.go
- No global state; all deps injected via constructor params
- main.go is the only wiring location
- Module `Register(router, deps)` pattern — no direct coupling between modules
- Error envelope: all errors use `{"error":{"code":"...","message":"...","fields":{...}}}`
- PATCH endpoints truly partial (no server-required fields from client)
- Responses nest objects (never return bare foreign-key IDs)
- `created_by_user_id` always server-set from JWT claims
- `RETURNING` clauses on all INSERT/UPDATE (no extra SELECT after write)
- Context propagated through all layers (no `context.Background()` inside request path)
- Worker `Dispatch` non-blocking; goroutines handle `ctx.Done()`
- ERD PK types match implementation (UUID v7, not INT)
- Denormalized FKs (`enrollment.course_id`, `schedule.course_id`) enforced in service layer

### 3. Performance Engineer

- N+1 queries in list endpoints (use JOIN or batch fetch, not per-row loop)
- All filter/sort columns have indexes (check: batches, enrollments, schedules, forms, events)
- `enrolled_count` aggregate uses indexed `(batch_id, status)` covering index
- Pagination enforced on all list endpoints; `per_page` capped at 100
- pgxpool `MaxConns` configured, not left at default
- Bulk insert via `unnest` for form answers (not per-row INSERT loop)
- No repeated DB calls for data already fetched in the same request
- `respondents.email` has unique index (needed for upsert ON CONFLICT)
- CSV export uses streaming (`http.Flusher`) not full in-memory buffer
- No `SELECT *` — sqlc should generate specific column lists
- Advisory lock maps UUID batch_id to `int64` correctly (`pg_advisory_xact_lock` takes bigint)

### 4. Reliability Engineer

- Multi-step operations wrapped in single pgx transaction (respondent upsert + response insert + answers bulk insert)
- Enrollment advisory lock: `pg_advisory_xact_lock(batch_id_as_int64)` inside serializable tx
- Serialization failure (40001) caught and retried once; retry uses backoff not immediate
- Unique violation (23505) mapped to 409 CONFLICT with correct error code
- Graceful shutdown: `signal.NotifyContext` + `http.Server.Shutdown(ctx)` with 30s timeout
- Context cancellation propagates through all DB calls (pgx propagates automatically)
- Worker pool goroutines exit on `ctx.Done()` (no goroutine leak)
- Sentry captures 500 errors (not 4xx); dropped worker jobs logged at WARN
- State machines enforced: `ValidateBatchTransition`, `ValidateFormTransition` functions exist
- Idempotent endpoints (publish, close) return 200 if already in target state
- Users/students never hard deleted; soft-delete strategy per entity matches spec
- `created_at`/`created_by` immutable after insert (no UPDATE touching these fields)
- Password reset token table exists in schema; tokens have expiry enforced at DB and app layer

### 5. Scalability Engineer

- Indexes present for all filter columns (full list in CLAUDE.md)
- No full-table scans (all WHERE clauses on indexed columns or with LIMIT)
- Stateless request handling (no server-side session; all state in DB)
- pgxpool not exhausted at <100 concurrent users with MaxConns=20
- Worker channel capacity bounded and reasonable for expected event rate
- Advisory locks scoped to minimum (batch_id per enrollment, not global)
- All pgx.Tx committed or rolled back in all code paths (no connection leak)
- No unbounded in-memory collections (maps/slices that grow with requests)
- Multi-instance deployment risk documented: worker pool must be externalized before scaling horizontally

## Dispatch Instructions

Run **5 refinement rounds**. Each round: 5 specialized agents run in parallel, then a discussion phase synthesizes cross-agent findings and produces targeted feedback for the next round. Quality improves with each pass.

---

### Round 1 — Initial Pass

**REQUIRED:** Dispatch all 5 agents simultaneously using `superpowers:dispatching-parallel-agents`.

Prompt template for each agent:

```
You are a [ROLE] reviewing a Go 1.25 HTTP API. This is Round 1 (initial pass).

Stack: chi router, pgx/v5 + pgxpool, sqlc, JWT RS256, bcrypt cost 12, slog, Sentry, channel-based worker pool. Modular monolith: internal/module/[name]/ has handler.go, service.go, repository.go per module.

Your EXCLUSIVE focus: [paste the relevant checklist section from this skill]

Directories to review:
- internal/module/ — all modules
- internal/middleware/ — auth, logging, recovery, CORS
- cmd/server/main.go — DI wiring
- internal/db/query/ — sqlc queries
- internal/db/migrations/ — schema
- internal/worker/ — worker pool

For each finding use exactly:
SEVERITY: CRITICAL | HIGH | MEDIUM | LOW
FILE: path/to/file.go:line
ISSUE: what is wrong and why it matters
FIX: specific, actionable fix

Only report real bugs, missing checks, or wrong patterns. No style issues.
```

After all 5 finish: collect all findings into a **Round 1 Findings Block** (grouped by role).

---

### Discussion Phase (runs after each round, before the next)

After each round, run a **Discussion Agent** — a single agent that receives all findings from the current round and produces a Discussion Report. This agent's output drives the next round's targeted prompts.

Discussion Agent prompt:

```
You are a Lead Reviewer synthesizing findings from 5 specialized agents reviewing a Go 1.25 HTTP API.

Below are the Round [N] findings from: Security Engineer, Backend Architect, Performance Engineer, Reliability Engineer, Scalability Engineer.

[PASTE ALL ROUND N FINDINGS]

Your tasks:
1. CROSS-DIMENSION CONFLICTS: Identify findings where two roles disagree or one role's fix breaks another's concern (e.g. a reliability fix that introduces a performance regression). List each conflict as: "CONFLICT: [Role A] finding at file:line vs [Role B] concern — resolution direction".

2. BLIND SPOTS: Identify code paths, edge cases, or system behaviors that NO agent flagged but that seem risky given the checklist. For each: "BLIND SPOT: [area] — [why it needs a second look] — [which role should investigate]".

3. AMPLIFICATIONS: Flag findings that are likely more severe than rated. For each: "AMPLIFY: file:line — originally [SEVERITY] — should be [SEVERITY] — reason".

4. DISMISSED FALSE POSITIVES: Flag findings that are likely wrong or non-issues given the stack. For each: "FALSE POSITIVE: file:line — reason".

5. ROUND [N+1] FOCUS AREAS: Produce a short directive per role (1-3 sentences) telling each agent what to specifically re-examine in the next round based on this discussion.

Output format:
## Conflicts
...
## Blind Spots
...
## Amplifications
...
## False Positives
...
## Next Round Directives
Security: ...
Architecture: ...
Performance: ...
Reliability: ...
Scalability: ...
```

---

### Rounds 2–5 — Refinement Passes

For each subsequent round, dispatch all 5 agents simultaneously. Each agent receives:

1. The original role checklist (unchanged)
2. Their specific directive from the Discussion Report
3. All prior findings from their role (to avoid duplication and to revisit flagged items)

Prompt template for rounds 2–5:

```
You are a [ROLE] reviewing a Go 1.25 HTTP API. This is Round [N] of 5.

Stack: chi router, pgx/v5 + pgxpool, sqlc, JWT RS256, bcrypt cost 12, slog, Sentry, channel-based worker pool.

Your EXCLUSIVE focus: [paste the relevant checklist section]

PRIOR FINDINGS FROM YOUR ROLE (do not re-report unless upgrading severity):
[paste prior round findings for this role]

DISCUSSION DIRECTIVE FOR THIS ROUND:
[paste this role's directive from the Discussion Report]

Tasks for this round:
- Re-examine the areas called out in your directive
- Investigate blind spots assigned to your role
- Revisit any finding the discussion flagged as a false positive — confirm or retract
- Look for NEW findings in under-examined areas
- Upgrade severity of amplified findings if confirmed

For each finding use exactly:
SEVERITY: CRITICAL | HIGH | MEDIUM | LOW
FILE: path/to/file.go:line
ISSUE: what is wrong and why it matters
FIX: specific, actionable fix
STATUS: NEW | CONFIRMED | UPGRADED | RETRACTED

Only report real bugs, missing checks, or wrong patterns. No style issues.
```

After all 5 finish: collect findings into a **Round N Findings Block**, then run Discussion Phase again (up to Round 4 discussion → Round 5 final pass).

---

### Round 5 — Final Pass

Same as Rounds 2–4 but agents also:

- Confirm every CRITICAL and HIGH finding is traceable to a specific file:line
- Mark any previously reported finding as RESOLVED if a prior-round fix would have addressed it
- Add FINAL CONFIDENCE: HIGH | MEDIUM | LOW per finding

After Round 5 agents finish, skip the Discussion Phase. Proceed directly to Final Synthesis.

---

## Final Synthesis

After Round 5 completes, consolidate all rounds into a single deduplicated output:

```
## Review Summary (5 Rounds Completed)

## Security [N findings]
CRITICAL: path:line — issue — FIX
HIGH: ...

## Architecture [N findings]
...

## Performance [N findings]
...

## Reliability [N findings]
...

## Scalability [N findings]
...

## Retracted Findings [N]
- path:line — originally [ROLE/SEVERITY] — retracted: reason

## Priority Action List (Top 10 cross-dimension, ordered by risk)
1. [SEVERITY] [Domain] path:line — specific fix
2. ...

## Iteration Delta
Round 1: N findings
Round 2: +N new, N upgraded, N retracted
Round 3: +N new, N upgraded, N retracted
Round 4: +N new, N upgraded, N retracted
Round 5: +N new, N upgraded, N retracted
Net: N confirmed findings
```

---

## Execution Checklist

| Step                    | Action                                           | Blocker if skipped                             |
| ----------------------- | ------------------------------------------------ | ---------------------------------------------- |
| Round 1 dispatch        | All 5 agents in parallel                         | Sequential run defeats purpose                 |
| Collect R1 findings     | Group by role into Findings Block                | Discussion agent needs full context            |
| Discussion Phase R1     | Single discussion agent, full R1 findings        | No cross-agent signal for R2                   |
| Rounds 2–5 dispatch     | All 5 agents in parallel, with directive         | Without directive, agents repeat R1            |
| Discussion Phases 2–4   | Run after R2, R3, R4; skip after R5              | Skip after R5 — go straight to Final Synthesis |
| Round 5 confidence tags | Agents tag FINAL CONFIDENCE per finding          | Final Synthesis quality degrades               |
| Final Synthesis         | Deduplicate, order by risk, show Iteration Delta | No actionable summary delivered                |

## Common Mistakes

| Mistake                                  | Fix                                                                      |
| ---------------------------------------- | ------------------------------------------------------------------------ |
| Running agents sequentially within round | Dispatch all 5 in parallel every round                                   |
| Skipping Discussion Phase                | Discussion is mandatory between rounds — it drives blind spot coverage   |
| Discussion agent gets partial findings   | Pass ALL findings from the round, all 5 roles                            |
| Agents re-report identical findings      | Prompt includes prior findings; STATUS field enforces differentiation    |
| Skipping Discussion after Round 5        | Round 5 goes directly to Final Synthesis — no Discussion Phase           |
| Reporting style/formatting issues        | Only real bugs, wrong patterns, missing enforcement                      |
| Generic advice ("add error handling")    | Every finding requires specific file:line                                |
| Not checking schema against spec         | Security and architecture agents must cross-check ERD vs implementation  |
| Stopping at Round 1                      | All 5 rounds are required; early stopping misses cross-agent blind spots |
