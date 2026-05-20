# ERD: Teman Berbahasa : Tutor Place Management System

---

## Entities & Attributes

### STUDENT

| Column              | Type    | Constraint                        |
| ------------------- | ------- | --------------------------------- |
| `student_id`        | INT     | PK                                |
| `first_name`        | VARCHAR | NOT NULL                          |
| `last_name`         | VARCHAR | NOT NULL                          |
| `email`             | VARCHAR | UNIQUE, NOT NULL                  |
| `phone`             | VARCHAR |                                   |
| `date_of_birth`     | DATE    |                                   |
| `gender`            | ENUM    | `male`, `female`, `other`         |
| `address`           | TEXT    |                                   |
| `parent_name`       | VARCHAR |                                   |
| `parent_phone`      | VARCHAR |                                   |
| `registration_date` | DATE    | NOT NULL                          |
| `status`            | ENUM    | `active`, `inactive`, `graduated` |

---

### USER

> System users who manage or teach in the platform (e.g. admins, teachers).

| Column          | Type     | Constraint                  |
| --------------- | -------- | --------------------------- |
| `user_id`       | INT      | PK                          |
| `first_name`    | VARCHAR  | NOT NULL                    |
| `last_name`     | VARCHAR  | NOT NULL                    |
| `email`         | VARCHAR  | UNIQUE, NOT NULL            |
| `password_hash` | VARCHAR  | NOT NULL                    |
| `role`          | ENUM     | `admin`, `teacher`, `staff` |
| `phone`         | VARCHAR  |                             |
| `status`        | ENUM     | `active`, `inactive`        |
| `created_at`    | DATETIME |                             |

---

### COURSE

| Column           | Type    | Constraint                             |
| ---------------- | ------- | -------------------------------------- |
| `course_id`      | INT     | PK                                     |
| `course_name`    | VARCHAR | NOT NULL                               |
| `course_code`    | VARCHAR | UNIQUE, NOT NULL                       |
| `description`    | TEXT    |                                        |
| `subject`        | VARCHAR |                                        |
| `level`          | ENUM    | `beginner`, `intermediate`, `advanced` |
| `session_count`  | INT     |                                        |
| `price`          | DECIMAL |                                        |
| `max_capacity`   | INT     |                                        |
| `status`         | ENUM    | `active`, `archived`                   |

---

### BATCH

> Each batch belongs to exactly **one course**, so batch numbering can be scoped per course (e.g. Math Batch 1, English Batch 1 are independent).

| Column               | Type    | Constraint                                        |
| -------------------- | ------- | ------------------------------------------------- |
| `batch_id`           | INT     | PK                                                |
| `course_id`          | INT     | FK → COURSE, NOT NULL                             |
| `instructor_user_id` | INT     | FK → USER — the default instructor for this batch |
| `created_by_user_id` | INT     | FK → USER — who created this batch                |
| `batch_name`         | VARCHAR | NOT NULL                                          |
| `batch_code`         | VARCHAR | NOT NULL                                          |
| `start_date`         | DATE    |                                                   |
| `end_date`           | DATE    |                                                   |
| `academic_year`      | VARCHAR |                                                   |
| `status`             | ENUM    | `upcoming`, `ongoing`, `completed`                |

> **Unique constraint:** `(course_id, batch_code)` — batch codes are unique per course, not globally.

---

### ENROLLMENT

> Resolves the **M:N** relationship between STUDENT and BATCH (which already implies the COURSE). `course_id` is kept as a **denormalized convenience FK** for direct joins — it must always match `batch.course_id`.

| Column            | Type    | Constraint                                               |
| ----------------- | ------- | -------------------------------------------------------- |
| `enrollment_id`   | INT     | PK                                                       |
| `student_id`      | INT     | FK → STUDENT                                             |
| `batch_id`        | INT     | FK → BATCH                                               |
| `course_id`       | INT     | FK → COURSE — denormalized; must match `batch.course_id` |
| `enrollment_date` | DATE    | NOT NULL                                                 |
| `status`          | ENUM    | `enrolled`, `dropped`, `completed`                       |
| `payment_status`  | ENUM    | `paid`, `pending`, `partial`                             |
| `final_grade`     | VARCHAR |                                                          |

> **Unique constraint:** `(student_id, batch_id)` — a student can only be in a batch once (course is implied by batch).

---

### SCHEDULE

> Defines the **recurring timetable** for a batch. `instructor_user_id` overrides the batch default for this slot — if `NULL`, the batch's default instructor applies.

| Column               | Type    | Constraint                                                        |
| -------------------- | ------- | ----------------------------------------------------------------- |
| `schedule_id`        | INT     | PK                                                                |
| `batch_id`           | INT     | FK → BATCH                                                        |
| `course_id`          | INT     | FK → COURSE — denormalized; must match `batch.course_id`          |
| `instructor_user_id` | INT     | FK → USER, nullable — overrides `batch.instructor_user_id` if set |
| `day_of_week`        | ENUM    | `monday` … `sunday`                                               |
| `start_time`         | TIME    |                                                                   |
| `end_time`           | TIME    |                                                                   |
| `room`               | VARCHAR |                                                                   |
| `recurrence`         | ENUM    | `weekly`, `one-time`                                              |
| `effective_from`     | DATE    |                                                                   |
| `effective_until`    | DATE    |                                                                   |

---

### SCHEDULE_OVERRIDE

> Handles **one-off exceptions** to a recurring schedule — a single session that is rescheduled, cancelled, or reassigned to a different instructor. Keeps the base SCHEDULE clean and exceptions fully auditable.

| Column                   | Type     | Constraint                                          |
| ------------------------ | -------- | --------------------------------------------------- |
| `override_id`            | INT      | PK                                                  |
| `schedule_id`            | INT      | FK → SCHEDULE — the recurring slot being overridden |
| `original_date`          | DATE     | NOT NULL — the specific session date being affected |
| `override_type`          | ENUM     | `reschedule`, `cancellation`, `instructor_change`   |
| `new_date`               | DATE     | nullable — set if rescheduled to a different day    |
| `new_start_time`         | TIME     | nullable — set if time changes                      |
| `new_end_time`           | TIME     | nullable — set if time changes                      |
| `new_room`               | VARCHAR  | nullable — set if room changes                      |
| `new_instructor_user_id` | INT      | FK → USER, nullable — replacement instructor        |
| `reason`                 | TEXT     | nullable                                            |
| `created_by_user_id`     | INT      | FK → USER — who logged the override                 |
| `created_at`             | DATETIME |                                                     |

---

### EVENT

> Standalone calendar entries, optionally targeted at specific audiences.

| Column           | Type     | Constraint                                   |
| ---------------- | -------- | -------------------------------------------- |
| `event_id`       | INT      | PK                                           |
| `title`          | VARCHAR  | NOT NULL                                     |
| `description`    | TEXT     |                                              |
| `event_type`     | ENUM     | `workshop`, `exam`, `holiday`, `meeting`     |
| `start_datetime` | DATETIME |                                              |
| `end_datetime`   | DATETIME |                                              |
| `location`       | VARCHAR  |                                              |
| `audience`       | ENUM     | `all`, `students`, `staff`, `specific_batch` |
| `created_by`     | INT      |                                              |
| `created_at`     | DATETIME |                                              |

---

### FORM

> Supports full lifecycle: **create → publish → close → delete** (soft delete via `deleted_at`).

| Column            | Type     | Constraint                                |
| ----------------- | -------- | ----------------------------------------- |
| `form_id`         | INT      | PK                                        |
| `title`           | VARCHAR  | NOT NULL                                  |
| `description`     | TEXT     |                                           |
| `status`          | ENUM     | `draft`, `published`, `closed`, `deleted` |
| `allow_anonymous` | BOOLEAN  | DEFAULT false                             |
| `created_by`      | INT      |                                           |
| `created_at`      | DATETIME |                                           |
| `published_at`    | DATETIME | nullable                                  |
| `closed_at`       | DATETIME | nullable                                  |
| `deleted_at`      | DATETIME | nullable — soft delete                    |

---

### FORM_QUESTION

| Column          | Type    | Constraint                                              |
| --------------- | ------- | ------------------------------------------------------- |
| `question_id`   | INT     | PK                                                      |
| `form_id`       | INT     | FK → FORM                                               |
| `question_text` | TEXT    | NOT NULL                                                |
| `question_type` | ENUM    | `text`, `multiple_choice`, `checkbox`, `rating`, `date` |
| `is_required`   | BOOLEAN | DEFAULT false                                           |
| `order_index`   | INT     |                                                         |
| `options`       | JSON    | nullable — for choice-type questions                    |

---

### RESPONDENT

> Decoupled from STUDENT to allow non-student respondents (e.g. parents, prospects).

| Column          | Type     | Constraint             |
| --------------- | -------- | ---------------------- |
| `respondent_id` | INT      | PK                     |
| `student_id`    | INT      | FK → STUDENT, nullable |
| `name`          | VARCHAR  |                        |
| `email`         | VARCHAR  |                        |
| `submitted_at`  | DATETIME |                        |

---

### FORM_RESPONSE

> One submission of a published form by a respondent.

| Column          | Type     | Constraint      |
| --------------- | -------- | --------------- |
| `response_id`   | INT      | PK              |
| `form_id`       | INT      | FK → FORM       |
| `respondent_id` | INT      | FK → RESPONDENT |
| `submitted_at`  | DATETIME |                 |

---

### FORM_ANSWER

> One answer to one question, linked to a specific response.

| Column         | Type | Constraint                                        |
| -------------- | ---- | ------------------------------------------------- |
| `answer_id`    | INT  | PK                                                |
| `response_id`  | INT  | FK → FORM_RESPONSE                                |
| `question_id`  | INT  | FK → FORM_QUESTION                                |
| `answer_text`  | TEXT | nullable                                          |
| `answer_value` | JSON | nullable — for multi-select or structured answers |

---

## Relationships

```
USER            1 ──────< M   BATCH   (as teacher_user_id)
USER            1 ──────< M   BATCH   (as created_by_user_id)
COURSE          1 ──────< M   BATCH

STUDENT         1 ──────< M   ENROLLMENT
BATCH           1 ──────< M   ENROLLMENT
COURSE          1 ──────< M   ENROLLMENT  (denormalized FK — mirrors batch.course_id)

BATCH           1 ──────< M   SCHEDULE
COURSE          1 ──────< M   SCHEDULE    (denormalized FK — mirrors batch.course_id)

FORM            1 ──────< M   FORM_QUESTION
FORM            1 ──────< M   FORM_RESPONSE
RESPONDENT      1 ──────< M   FORM_RESPONSE
FORM_RESPONSE   1 ──────< M   FORM_ANSWER
FORM_QUESTION   1 ──────< M   FORM_ANSWER

STUDENT         1 ──────< M   RESPONDENT  (optional / nullable)

EVENT           — standalone (no hard FK; audience is an enum field)
```

---

## Cardinality Summary

| Relationship             | Cardinality | Resolved Via                                      |
| ------------------------ | ----------- | ------------------------------------------------- |
| Student ↔ Batch (Course) | M : N       | `ENROLLMENT`                                      |
| Course ↔ Batch           | 1 : M       | `BATCH.course_id` (a batch belongs to one course) |
| User (Teacher) ↔ Batch   | 1 : M       | `BATCH.teacher_user_id`                           |
| Form ↔ Respondent        | M : N       | `FORM_RESPONSE`                                   |
| Form Question ↔ Response | M : N       | `FORM_ANSWER`                                     |

---

## Design Notes

- **BATCH now owns `course_id`** — making the relationship 1 COURSE : M BATCHES. Batch codes are unique per course (not globally), so Math can have Batch 1 and English can also have Batch 1 independently. The unique constraint is `(course_id, batch_code)`.
- **USER is the system user** (admins, teachers, staff). BATCH has two FK references to USER: `teacher_user_id` (the assigned teacher) and `created_by_user_id` (audit trail). The teacher is modeled on the batch, not on individual schedules, since one teacher typically owns a whole batch.
- **ENROLLMENT keeps `course_id`** as a denormalized convenience column, since `batch_id` already implies the course. This makes direct joins to COURSE possible without routing through BATCH, at the cost of needing to ensure consistency (enforce via DB constraint or app logic that `enrollment.course_id = batch.course_id`). The unique constraint simplifies to `(student_id, batch_id)`.
- **SCHEDULE also keeps `course_id`** for the same denormalization reason, consistent with ENROLLMENT. The `instructor` field was removed since the teacher is now captured on BATCH.
- **FORM lifecycle** is managed through the `status` field (`draft → published → closed → deleted`) and nullable timestamps (`published_at`, `closed_at`, `deleted_at`). Soft deletes via `deleted_at` preserve historical response data.
- **FORM_ANSWER** normalizes all answers to one row per question per submission, supporting any question type cleanly. Structured answers (multi-select, ratings) use the `answer_value` JSON column.
- **RESPONDENT** is decoupled from STUDENT so non-students can respond to published forms, with an optional FK back to STUDENT when the respondent is known.
- **EVENT** is intentionally standalone. If future requirements call for linking events to specific batches or students, an `EVENT_AUDIENCE` join table would be the natural extension.
