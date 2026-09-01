# ERD: Teman Berbahasa : Tutor Place Management System

> Reflects the shipped schema (`internal/db/migrations/001`–`013`). For API-payload
> shapes see `data-contract.md`; that document wins for JSON, this one wins for DB
> constraints.

**Conventions**

- All PKs are `UUID` (v7, time-ordered) — never integers.
- All timestamps are `TIMESTAMPTZ` stored in UTC. `day_of_week` is interpreted in WIB (UTC+7).
- `TIME` columns carry no timezone and are WIB.
- Money is `NUMERIC(12,2)` — fixed-point, never float.
- Structured columns are `JSONB`.
- User foreign keys follow the `<role>_user_id` convention: `instructor_user_id`,
  `new_instructor_user_id`, `creator_user_id`.

---

## Entities & Attributes

### USER

> System users who manage or teach in the platform (admins, teachers, staff).

| Column            | Type         | Constraint                                                                                    |
| ----------------- | ------------ | --------------------------------------------------------------------------------------------- |
| `id`              | UUID         | PK                                                                                            |
| `first_name`      | VARCHAR(100) | NOT NULL                                                                                      |
| `last_name`       | VARCHAR(100) | NOT NULL                                                                                      |
| `email`           | VARCHAR(255) | UNIQUE, NOT NULL                                                                              |
| `password_hash`   | VARCHAR(255) | NOT NULL — bcrypt cost 12                                                                     |
| `role`            | ENUM         | `admin`, `teacher`, `staff` — NOT NULL                                                        |
| `phone`           | VARCHAR(20)  |                                                                                               |
| `status`          | ENUM         | `active`, `inactive` — NOT NULL, default `active`                                             |
| `failed_attempts` | INT          | NOT NULL, default `0` — consecutive failed logins; account locks at 10, reset to 0 on success |
| `created_at`      | TIMESTAMPTZ  | NOT NULL, default now()                                                                       |
| `updated_at`      | TIMESTAMPTZ  | NOT NULL, default now()                                                                       |

---

### REFRESH_TOKEN

> One row per active session. Stores only `SHA-256(raw_token)` — the raw token never persists. Hard-deleted on logout / rotation; expired rows purged by cron.

| Column       | Type        | Constraint                      |
| ------------ | ----------- | ------------------------------- |
| `id`         | UUID        | PK                              |
| `user_id`    | UUID        | FK → USER, NOT NULL             |
| `token_hash` | VARCHAR(64) | UNIQUE, NOT NULL                |
| `expires_at` | TIMESTAMPTZ | NOT NULL — 7 days from issuance |
| `created_at` | TIMESTAMPTZ | NOT NULL, default now()         |

---

### PASSWORD_RESET_TOKEN

> Same hash-only pattern as `REFRESH_TOKEN`; short-lived.

| Column       | Type        | Constraint              |
| ------------ | ----------- | ----------------------- |
| `id`         | UUID        | PK                      |
| `user_id`    | UUID        | FK → USER, NOT NULL     |
| `token_hash` | VARCHAR(64) | UNIQUE, NOT NULL        |
| `expires_at` | TIMESTAMPTZ | NOT NULL                |
| `created_at` | TIMESTAMPTZ | NOT NULL, default now() |

---

### STUDENT

| Column              | Type         | Constraint                                                     |
| ------------------- | ------------ | -------------------------------------------------------------- |
| `id`                | UUID         | PK                                                             |
| `first_name`        | VARCHAR(100) | NOT NULL                                                       |
| `last_name`         | VARCHAR(100) | NOT NULL                                                       |
| `email`             | VARCHAR(255) | UNIQUE, NOT NULL                                               |
| `phone`             | VARCHAR(20)  |                                                                |
| `date_of_birth`     | DATE         |                                                                |
| `gender`            | ENUM         | `male`, `female`, `other`                                      |
| `address`           | TEXT         |                                                                |
| `parent_name`       | VARCHAR(200) |                                                                |
| `parent_phone`      | VARCHAR(20)  |                                                                |
| `registration_date` | DATE         | NOT NULL, default `CURRENT_DATE`                               |
| `status`            | ENUM         | `active`, `inactive`, `graduated` — NOT NULL, default `active` |
| `created_at`        | TIMESTAMPTZ  | NOT NULL, default now()                                        |
| `updated_at`        | TIMESTAMPTZ  | NOT NULL, default now()                                        |

---

### COURSE

| Column          | Type          | Constraint                                                 |
| --------------- | ------------- | ---------------------------------------------------------- |
| `id`            | UUID          | PK                                                         |
| `course_name`   | VARCHAR(200)  | NOT NULL                                                   |
| `course_code`   | VARCHAR(20)   | UNIQUE, NOT NULL                                           |
| `description`   | TEXT          |                                                            |
| `subject`       | VARCHAR(100)  |                                                            |
| `level`         | ENUM          | `beginner`, `intermediate`, `advanced`                     |
| `session_count` | INT           | NOT NULL — max scheduled sessions per batch of this course |
| `price`         | NUMERIC(12,2) |                                                            |
| `max_capacity`  | INT           | max students per batch — enforced at enrollment            |
| `status`        | ENUM          | `active`, `archived` — NOT NULL, default `active`          |
| `created_at`    | TIMESTAMPTZ   | NOT NULL, default now()                                    |
| `updated_at`    | TIMESTAMPTZ   | NOT NULL, default now()                                    |

---

### BATCH

> Each batch belongs to exactly **one course**, so batch numbering is scoped per course (Math Batch 1 and English Batch 1 are independent).

| Column               | Type         | Constraint                                                               |
| -------------------- | ------------ | ------------------------------------------------------------------------ |
| `id`                 | UUID         | PK                                                                       |
| `course_id`          | UUID         | FK → COURSE, NOT NULL                                                    |
| `instructor_user_id` | UUID         | FK → USER (role = teacher), NOT NULL — default instructor for this batch |
| `creator_user_id`    | UUID         | FK → USER, NOT NULL — audit trail; server-set from JWT                   |
| `batch_name`         | VARCHAR(200) | NOT NULL                                                                 |
| `batch_code`         | VARCHAR(20)  | NOT NULL                                                                 |
| `academic_year`      | VARCHAR(10)  | administrative label, e.g. `"2026"`                                      |
| `status`             | ENUM         | `upcoming`, `ongoing`, `completed` — NOT NULL, default `upcoming`        |
| `created_at`         | TIMESTAMPTZ  | NOT NULL, default now()                                                  |
| `updated_at`         | TIMESTAMPTZ  | NOT NULL, default now()                                                  |

> **Unique constraint:** `(course_id, batch_code)` — batch codes are unique per course, not globally.
> **No `start_date` / `end_date`** (dropped in migration 005). The date range is derived
> from schedules; the `batches_with_stats` view exposes `first_class_date` / `last_class_date`
> as `MIN`/`MAX` of schedule effective dates.

---

### ENROLLMENT

> Resolves the **M:N** relationship between STUDENT and BATCH (which already implies the COURSE). `course_id` is a **denormalized convenience FK** — it must always match `batch.course_id` (server-enforced).

| Column            | Type        | Constraint                                                         |
| ----------------- | ----------- | ------------------------------------------------------------------ |
| `id`              | UUID        | PK                                                                 |
| `student_id`      | UUID        | FK → STUDENT, NOT NULL                                             |
| `batch_id`        | UUID        | FK → BATCH, NOT NULL                                               |
| `course_id`       | UUID        | FK → COURSE, NOT NULL — denormalized; must match `batch.course_id` |
| `enrollment_date` | DATE        | NOT NULL, default `CURRENT_DATE`                                   |
| `status`          | ENUM        | `enrolled`, `dropped`, `completed` — NOT NULL, default `enrolled`  |
| `payment_status`  | ENUM        | `pending`, `partial`, `paid` — NOT NULL, default `pending`         |
| `final_grade`     | VARCHAR(10) | free-text, only meaningful when `status = completed`               |
| `created_at`      | TIMESTAMPTZ | NOT NULL, default now()                                            |
| `updated_at`      | TIMESTAMPTZ | NOT NULL, default now()                                            |

> **Unique constraint:** `(student_id, batch_id)` — a student can only be in a batch once.

---

### SCHEDULE

> Defines the **recurring timetable** for a batch. `instructor_user_id` overrides the batch default for this slot — if `NULL`, the batch's default instructor applies.

| Column               | Type         | Constraint                                                         |
| -------------------- | ------------ | ------------------------------------------------------------------ |
| `id`                 | UUID         | PK                                                                 |
| `batch_id`           | UUID         | FK → BATCH, NOT NULL                                               |
| `course_id`          | UUID         | FK → COURSE, NOT NULL — denormalized; must match `batch.course_id` |
| `instructor_user_id` | UUID         | FK → USER, **nullable** — slot-level instructor override           |
| `day_of_week`        | ENUM         | `monday` … `sunday` — NOT NULL                                     |
| `start_time`         | TIME         | NOT NULL                                                           |
| `end_time`           | TIME         | NOT NULL — must be after `start_time`                              |
| `room`               | VARCHAR(100) |                                                                    |
| `recurrence`         | ENUM         | `weekly`, `one-time` — NOT NULL                                    |
| `effective_from`     | DATE         | NOT NULL — for `one-time`, the single session date                 |
| `effective_until`    | DATE         | **nullable** — `NULL` = open-ended                                 |
| `created_at`         | TIMESTAMPTZ  | NOT NULL, default now()                                            |
| `updated_at`         | TIMESTAMPTZ  | NOT NULL, default now()                                            |

---

### SCHEDULE_OVERRIDE

> One-off exceptions to a recurring slot — a single session rescheduled or reassigned to a different instructor. Keeps the base SCHEDULE clean and exceptions auditable.

| Column                   | Type         | Constraint                                             |
| ------------------------ | ------------ | ------------------------------------------------------ |
| `id`                     | UUID         | PK                                                     |
| `schedule_id`            | UUID         | FK → SCHEDULE, NOT NULL, **ON DELETE CASCADE**         |
| `original_date`          | DATE         | NOT NULL — the specific session date being affected    |
| `override_type`          | ENUM         | `reschedule`, `instructor_change` — NOT NULL           |
| `new_date`               | DATE         | nullable — required when `override_type = reschedule`  |
| `new_start_time`         | TIME         | nullable — only meaningful for `reschedule`            |
| `new_end_time`           | TIME         | nullable — only meaningful for `reschedule`            |
| `new_room`               | VARCHAR(100) | nullable — only meaningful for `reschedule`            |
| `new_instructor_user_id` | UUID         | FK → USER, nullable — required for `instructor_change` |
| `reason`                 | TEXT         | nullable                                               |
| `creator_user_id`        | UUID         | FK → USER, NOT NULL — server-set from JWT              |
| `created_at`             | TIMESTAMPTZ  | NOT NULL, default now()                                |

> **Unique constraint:** `(schedule_id, original_date)` — one override per session per slot.
> No `updated_at` — overrides are replaced (upserted on the unique key) or hard-deleted.

---

### EVENT

> Institution-wide calendar entries, optionally targeted at an audience.

| Column            | Type         | Constraint                                              |
| ----------------- | ------------ | ------------------------------------------------------- |
| `id`              | UUID         | PK                                                      |
| `title`           | VARCHAR(200) | NOT NULL                                                |
| `description`     | TEXT         |                                                         |
| `event_type`      | ENUM         | `workshop`, `exam`, `holiday`, `meeting` — NOT NULL     |
| `start_datetime`  | TIMESTAMPTZ  | NOT NULL                                                |
| `end_datetime`    | TIMESTAMPTZ  | NOT NULL — must be after `start_datetime`               |
| `location`        | VARCHAR(200) |                                                         |
| `audience`        | ENUM         | `all`, `students`, `staff`, `specific_batch` — NOT NULL |
| `creator_user_id` | UUID         | FK → USER, NOT NULL — server-set from JWT               |
| `created_at`      | TIMESTAMPTZ  | NOT NULL, default now()                                 |
| `updated_at`      | TIMESTAMPTZ  | NOT NULL, default now()                                 |

> `audience = specific_batch` is a flag only. Actual batch linking would be a future
> `event_audience` join table.

---

### FORM

> Full lifecycle: **draft → published → closed → deleted** (soft delete via `deleted_at`).

| Column            | Type         | Constraint                                                            |
| ----------------- | ------------ | --------------------------------------------------------------------- |
| `id`              | UUID         | PK                                                                    |
| `title`           | VARCHAR(200) | NOT NULL                                                              |
| `description`     | TEXT         |                                                                       |
| `status`          | ENUM         | `draft`, `published`, `closed`, `deleted` — NOT NULL, default `draft` |
| `allow_anonymous` | BOOLEAN      | NOT NULL, default `false`                                             |
| `creator_user_id` | UUID         | FK → USER, NOT NULL — server-set from JWT                             |
| `created_at`      | TIMESTAMPTZ  | NOT NULL, default now()                                               |
| `updated_at`      | TIMESTAMPTZ  | NOT NULL, default now()                                               |
| `published_at`    | TIMESTAMPTZ  | nullable                                                              |
| `closed_at`       | TIMESTAMPTZ  | nullable                                                              |
| `deleted_at`      | TIMESTAMPTZ  | nullable — soft delete                                                |

---

### FORM_QUESTION

| Column          | Type    | Constraint                                                         |
| --------------- | ------- | ------------------------------------------------------------------ |
| `id`            | UUID    | PK                                                                 |
| `form_id`       | UUID    | FK → FORM, NOT NULL                                                |
| `question_text` | TEXT    | NOT NULL                                                           |
| `question_type` | ENUM    | `text`, `multiple_choice`, `checkbox`, `rating`, `date` — NOT NULL |
| `is_required`   | BOOLEAN | NOT NULL, default `false`                                          |
| `order_index`   | INT     | NOT NULL — 1-based display order                                   |
| `options`       | JSONB   | nullable — array of option strings; only for choice types          |

> **Immutability:** questions cannot be created, edited, or deleted once the parent form is `published`.

---

### RESPONDENT

> Decoupled from STUDENT so non-students (parents, prospects) can respond. Upserted on `email` — the same email across multiple forms reuses one row.

| Column       | Type         | Constraint                                              |
| ------------ | ------------ | ------------------------------------------------------- |
| `id`         | UUID         | PK                                                      |
| `student_id` | UUID         | FK → STUDENT, nullable                                  |
| `name`       | VARCHAR(200) | nullable — required when `form.allow_anonymous = false` |
| `email`      | VARCHAR(255) | nullable — required when `form.allow_anonymous = false` |
| `created_at` | TIMESTAMPTZ  | NOT NULL, default now()                                 |

> **Partial unique index:** `UNIQUE (email) WHERE email IS NOT NULL` (migration 009).
> A _partial_ index only enforces uniqueness over the rows matching its `WHERE` clause —
> here, rows with an email. Anonymous respondents (email `NULL`) are exempt, so many
> anonymous rows can coexist.

---

### FORM_RESPONSE

> One submission of a form by a respondent.

| Column          | Type        | Constraint                |
| --------------- | ----------- | ------------------------- |
| `id`            | UUID        | PK                        |
| `form_id`       | UUID        | FK → FORM, NOT NULL       |
| `respondent_id` | UUID        | FK → RESPONDENT, NOT NULL |
| `submitted_at`  | TIMESTAMPTZ | NOT NULL, default now()   |

> **Unique index:** `(form_id, respondent_id)`. For identified forms the respondent is
> upserted by email, so this blocks a second submission. For anonymous forms each
> submission creates a fresh respondent row, so the pair never collides.

---

### FORM_ANSWER

> One answer to one question within a response.

| Column         | Type  | Constraint                                                                          |
| -------------- | ----- | ----------------------------------------------------------------------------------- |
| `id`           | UUID  | PK                                                                                  |
| `response_id`  | UUID  | FK → FORM_RESPONSE, NOT NULL                                                        |
| `question_id`  | UUID  | FK → FORM_QUESTION, NOT NULL — validated that `question.form_id = response.form_id` |
| `answer_text`  | TEXT  | nullable — for `text` and `date`                                                    |
| `answer_value` | JSONB | nullable — for `multiple_choice`, `checkbox`, `rating`                              |

---

## Relationships

```
USER            1 ──────< M   REFRESH_TOKEN
USER            1 ──────< M   PASSWORD_RESET_TOKEN

USER            1 ──────< M   BATCH              (as instructor_user_id)
USER            1 ──────< M   BATCH              (as creator_user_id)
COURSE          1 ──────< M   BATCH

STUDENT         1 ──────< M   ENROLLMENT
BATCH           1 ──────< M   ENROLLMENT
COURSE          1 ──────< M   ENROLLMENT         (denormalized FK — mirrors batch.course_id)

BATCH           1 ──────< M   SCHEDULE
COURSE          1 ──────< M   SCHEDULE           (denormalized FK — mirrors batch.course_id)
USER            1 ──────< M   SCHEDULE           (as instructor_user_id, nullable)

SCHEDULE        1 ──────< M   SCHEDULE_OVERRIDE  (ON DELETE CASCADE)
USER            1 ──────< M   SCHEDULE_OVERRIDE  (as new_instructor_user_id, nullable)
USER            1 ──────< M   SCHEDULE_OVERRIDE  (as creator_user_id)

USER            1 ──────< M   EVENT              (as creator_user_id)

USER            1 ──────< M   FORM               (as creator_user_id)
FORM            1 ──────< M   FORM_QUESTION
FORM            1 ──────< M   FORM_RESPONSE
RESPONDENT      1 ──────< M   FORM_RESPONSE
FORM_RESPONSE   1 ──────< M   FORM_ANSWER
FORM_QUESTION   1 ──────< M   FORM_ANSWER

STUDENT         1 ──────< M   RESPONDENT         (optional / nullable)
```

---

## Cardinality Summary

| Relationship             | Cardinality | Resolved Via                    |
| ------------------------ | ----------- | ------------------------------- |
| Student ↔ Batch (Course) | M : N       | `ENROLLMENT`                    |
| Course ↔ Batch           | 1 : M       | `BATCH.course_id`               |
| User (Teacher) ↔ Batch   | 1 : M       | `BATCH.instructor_user_id`      |
| User (Creator) ↔ Batch   | 1 : M       | `BATCH.creator_user_id`         |
| Schedule ↔ Override      | 1 : M       | `SCHEDULE_OVERRIDE.schedule_id` |
| Form ↔ Respondent        | M : N       | `FORM_RESPONSE`                 |
| Form Question ↔ Response | M : N       | `FORM_ANSWER`                   |

---

## Design Notes

- **BATCH owns `course_id`** — 1 COURSE : M BATCHES. Batch codes are unique per course
  (`(course_id, batch_code)`), so Math and English can each have "Batch 1".
- **USER is the system user** (admins, teachers, staff). BATCH has two FK references to
  USER: `instructor_user_id` (assigned teacher) and `creator_user_id` (audit). The teacher
  is modeled on the batch, not on individual schedules, since one teacher typically owns a
  whole batch — a SCHEDULE may still set `instructor_user_id` to override one slot.
- **`created_by_user_id` was renamed to `creator_user_id`** (migration 013) on BATCH,
  SCHEDULE_OVERRIDE, EVENT and FORM, to match the `<role>_user_id` convention shared with
  `instructor_user_id` / `new_instructor_user_id`.
- **ENROLLMENT keeps `course_id`** as a denormalized convenience column since `batch_id`
  already implies the course. Consistency (`enrollment.course_id = batch.course_id`) is
  enforced in app logic. The unique constraint simplifies to `(student_id, batch_id)`.
- **SCHEDULE also keeps `course_id`** for the same denormalization reason, consistent with
  ENROLLMENT. There is no name-only instructor field — the teacher is captured on BATCH,
  with an optional `instructor_user_id` slot override.
- **SCHEDULE_OVERRIDE** has `ON DELETE CASCADE` on `schedule_id`, so deleting a schedule
  removes its overrides. It has no `updated_at`: an override for a given `(schedule_id,
original_date)` is upserted, or hard-deleted.
- **FORM lifecycle** is managed through `status` (`draft → published → closed → deleted`)
  plus nullable timestamps (`published_at`, `closed_at`, `deleted_at`). Soft deletes
  preserve historical response data.
- **RESPONDENT** is decoupled from STUDENT so non-students can respond, with an optional
  FK back to STUDENT when the respondent is a known student.
- **EVENT** carries only a `creator_user_id` FK and an `audience` enum. Linking events to
  specific batches/students would be a future `event_audience` join table.
- **Auth token tables** (`REFRESH_TOKEN`, `PASSWORD_RESET_TOKEN`) store only
  `SHA-256(raw_token)`; raw tokens never touch the database.
