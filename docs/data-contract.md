# Teman Berbahasa : Data Dictionary & API Contract

## Tutor Place Management System

**Version:** 1.0  
**Audience:** Frontend Engineers, Backend Engineers  
**Purpose:** Single source of truth for (1) database table definitions with field-level rationale, and (2) exact JSON shapes for every API request and response. When the ERD and this document conflict, this document wins for API shapes; the ERD wins for DB constraints.

**Conventions:**

- All IDs: UUID v7 string — `"019687a2-1234-7abc-8def-000000000001"`
- All datetimes: ISO 8601, UTC — `"2025-05-18T07:00:00Z"`
- All dates (no time component): `"2025-05-18"`
- All times (no date): `"09:00:00"` (24-hour, HH:MM:SS)
- Nullable fields marked `null` in examples; absent and `null` treated identically on input
- `PATCH` requests are partial — only included fields are updated
- Field names: `snake_case` throughout
- Passwords never appear in any response payload
- `password_hash` never leaves the database

---

## Table of Contents

1. [Global Enums](#1-global-enums)
2. [Shared Response Objects](#2-shared-response-objects)
3. [Error Envelope](#3-error-envelope)
4. [Pagination Envelope](#4-pagination-envelope)
5. [Users](#5-users)
6. [Students](#6-students)
7. [Courses](#7-courses)
8. [Batches](#8-batches)
9. [Enrollments](#9-enrollments)
10. [Schedules](#10-schedules)
11. [Schedule Overrides](#11-schedule-overrides)
12. [Events](#12-events)
13. [Forms](#13-forms)
14. [Form Questions](#14-form-questions)
15. [Form Responses](#15-form-responses)
16. [Auth](#16-auth)

---

## 1. Global Enums

Enums are always transmitted as lowercase strings. Never send or expect integer codes.

### User

| Enum         | Values                          |
| ------------ | ------------------------------- |
| `UserRole`   | `"admin"` `"teacher"` `"staff"` |
| `UserStatus` | `"active"` `"inactive"`         |

### Student

| Enum            | Values                                |
| --------------- | ------------------------------------- |
| `StudentStatus` | `"active"` `"inactive"` `"graduated"` |
| `StudentGender` | `"male"` `"female"` `"other"`         |

### Course

| Enum           | Values                                     |
| -------------- | ------------------------------------------ |
| `CourseLevel`  | `"beginner"` `"intermediate"` `"advanced"` |
| `CourseStatus` | `"active"` `"archived"`                    |

### Batch

| Enum          | Values                                 |
| ------------- | -------------------------------------- |
| `BatchStatus` | `"upcoming"` `"ongoing"` `"completed"` |

### Enrollment

| Enum               | Values                                 |
| ------------------ | -------------------------------------- |
| `EnrollmentStatus` | `"enrolled"` `"dropped"` `"completed"` |
| `PaymentStatus`    | `"pending"` `"partial"` `"paid"`       |

### Schedule

| Enum                 | Values                                                                               |
| -------------------- | ------------------------------------------------------------------------------------ |
| `DayOfWeek`          | `"monday"` `"tuesday"` `"wednesday"` `"thursday"` `"friday"` `"saturday"` `"sunday"` |
| `ScheduleRecurrence` | `"weekly"` `"one-time"`                                                              |

### Schedule Override

| Enum           | Values                                                |
| -------------- | ----------------------------------------------------- |
| `OverrideType` | `"reschedule"` `"cancellation"` `"instructor_change"` |

### Event

| Enum            | Values                                            |
| --------------- | ------------------------------------------------- |
| `EventType`     | `"workshop"` `"exam"` `"holiday"` `"meeting"`     |
| `EventAudience` | `"all"` `"students"` `"staff"` `"specific_batch"` |

### Form

| Enum           | Values                                                        |
| -------------- | ------------------------------------------------------------- |
| `FormStatus`   | `"draft"` `"published"` `"closed"` `"deleted"`                |
| `QuestionType` | `"text"` `"multiple_choice"` `"checkbox"` `"rating"` `"date"` |

---

## 2. Shared Response Objects

Nested objects reused across multiple responses.

### `UserSummary`

Lightweight user reference embedded in other objects (e.g. instructor on a batch, created_by on a form).

```json
{
  "id": "019687a2-0001-7000-8000-000000000001",
  "first_name": "Budi",
  "last_name": "Santoso",
  "email": "budi@tutorplace.id",
  "role": "teacher"
}
```

### `CourseSummary`

Lightweight course reference embedded in batch and enrollment objects.

```json
{
  "id": "019687a2-0002-7000-8000-000000000001",
  "course_name": "Matematika Dasar",
  "course_code": "MTK01",
  "level": "beginner"
}
```

### `BatchSummary`

Lightweight batch reference embedded in enrollment and schedule objects.

```json
{
  "id": "019687a2-0003-7000-8000-000000000001",
  "batch_name": "Matematika Dasar Batch 3",
  "batch_code": "B003",
  "status": "ongoing"
}
```

### `StudentSummary`

Lightweight student reference embedded in enrollment objects.

```json
{
  "id": "019687a2-0004-7000-8000-000000000001",
  "first_name": "Rina",
  "last_name": "Kusuma",
  "email": "rina@email.com"
}
```

---

## 3. Error Envelope

All error responses use this shape regardless of status code.

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "fields": {
      "email": "must be a valid email address",
      "start_date": "must be before end_date"
    }
  }
}
```

| Field     | Notes                                                                                                       |
| --------- | ----------------------------------------------------------------------------------------------------------- |
| `code`    | Machine-readable error code string. Frontend uses this for conditional logic, not `message`.                |
| `message` | Human-readable summary. Do not display directly to end users — translate in frontend.                       |
| `fields`  | Present only on `422 Unprocessable Entity`. Map of `field_name → error string`. Absent on all other errors. |

### Standard error codes

| HTTP | `code`             | Meaning                                                            |
| ---- | ------------------ | ------------------------------------------------------------------ |
| 400  | `BAD_REQUEST`      | Malformed JSON or missing required body                            |
| 401  | `UNAUTHORIZED`     | Missing or invalid token                                           |
| 403  | `FORBIDDEN`        | Valid token but insufficient role                                  |
| 404  | `NOT_FOUND`        | Resource does not exist                                            |
| 409  | `CONFLICT`         | Duplicate — unique constraint would be violated                    |
| 410  | `GONE`             | Resource existed but is permanently unavailable (e.g. closed form) |
| 422  | `VALIDATION_ERROR` | Business rule or field validation failed; see `fields`             |
| 429  | `RATE_LIMITED`     | Too many requests; check `Retry-After` header                      |
| 500  | `INTERNAL_ERROR`   | Unexpected server error                                            |

---

## 4. Pagination Envelope

All `GET` list endpoints return this wrapper.

```json
{
  "data": [],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 143,
    "total_pages": 8
  }
}
```

**Query params for all list endpoints:**

| Param      | Default | Max   | Notes     |
| ---------- | ------- | ----- | --------- |
| `page`     | `1`     | —     | 1-indexed |
| `per_page` | `20`    | `100` |           |

---

## 5. Users

### DB Table: `users`

| Column            | Type         | Nullable | Rationale                                                               |
| ----------------- | ------------ | -------- | ----------------------------------------------------------------------- |
| `id`              | UUID v7      | No       | Primary key. UUID v7 for time-ordered index locality.                   |
| `first_name`      | VARCHAR(100) | No       | —                                                                       |
| `last_name`       | VARCHAR(100) | No       | —                                                                       |
| `email`           | VARCHAR(255) | No       | Login identifier. Globally unique.                                      |
| `password_hash`   | VARCHAR(255) | No       | bcrypt hash (cost 12). Never returned in any API response.              |
| `role`            | ENUM         | No       | Drives all permission checks. One role per user — no multi-role in v1.  |
| `phone`           | VARCHAR(20)  | Yes      | Contact info only; not used for auth.                                   |
| `status`          | ENUM         | No       | `inactive` is the soft-delete equivalent. Default `active`.             |
| `failed_attempts` | INT          | No       | Incremented on failed login. Reset on success. Lock at 10. Default `0`. |
| `created_at`      | TIMESTAMPTZ  | No       | Server-set at insert.                                                   |
| `updated_at`      | TIMESTAMPTZ  | No       | Server-set on every update.                                             |

**Relationships:**

- Referenced by `batches.instructor_user_id` and `batches.created_by_user_id`
- Referenced by `schedules.instructor_user_id` and `schedule_overrides.new_instructor_user_id`
- Referenced by `schedule_overrides.created_by_user_id`
- Referenced by `forms.created_by`
- Referenced by `events.created_by`

---

### `GET /users`

**Query params:** `role`, `status`, `page`, `per_page`

**Response `200`:**

```json
{
  "data": [
    {
      "id": "019687a2-0001-7000-8000-000000000001",
      "first_name": "Budi",
      "last_name": "Santoso",
      "email": "budi@tutorplace.id",
      "role": "teacher",
      "phone": "081234567890",
      "status": "active",
      "created_at": "2025-01-10T03:00:00Z",
      "updated_at": "2025-01-10T03:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 5,
    "total_pages": 1
  }
}
```

---

### `POST /users`

**Request:**

```json
{
  "first_name": "Budi",
  "last_name": "Santoso",
  "email": "budi@tutorplace.id",
  "password": "rahasia123",
  "role": "teacher",
  "phone": "081234567890"
}
```

**Response `201`:**

```json
{
  "id": "019687a2-0001-7000-8000-000000000001",
  "first_name": "Budi",
  "last_name": "Santoso",
  "email": "budi@tutorplace.id",
  "role": "teacher",
  "phone": "081234567890",
  "status": "active",
  "created_at": "2025-05-18T03:00:00Z",
  "updated_at": "2025-05-18T03:00:00Z"
}
```

---

### `GET /users/:id`

**Response `200`:** Same shape as single item in list above.

---

### `PATCH /users/:id`

**Request** (all fields optional):

```json
{
  "first_name": "Budi",
  "phone": "089999999999",
  "status": "inactive"
}
```

**Response `200`:** Full updated user object.

---

### `DELETE /users/:id`

Sets `status = inactive`. No body.

**Response `200`:**

```json
{
  "id": "019687a2-0001-7000-8000-000000000001",
  "status": "inactive"
}
```

---

## 6. Students

### DB Table: `students`

| Column              | Type         | Nullable | Rationale                                                                                                                       |
| ------------------- | ------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `id`                | UUID v7      | No       | PK                                                                                                                              |
| `first_name`        | VARCHAR(100) | No       | —                                                                                                                               |
| `last_name`         | VARCHAR(100) | No       | —                                                                                                                               |
| `email`             | VARCHAR(255) | Yes      | Unique when provided. Not required — some students (esp. younger) may not have email.                                           |
| `phone`             | VARCHAR(20)  | Yes      | Student's own phone.                                                                                                            |
| `date_of_birth`     | DATE         | Yes      | Used to derive age; not strictly required for enrollment.                                                                       |
| `gender`            | ENUM         | Yes      | Optional demographic field.                                                                                                     |
| `address`           | TEXT         | Yes      | Physical address for records.                                                                                                   |
| `parent_name`       | VARCHAR(200) | Yes      | Important for underage students; emergency contact.                                                                             |
| `parent_phone`      | VARCHAR(20)  | Yes      | Primary contact for parents/guardians.                                                                                          |
| `registration_date` | DATE         | No       | When the student was registered in the system. Server-set to today if not provided.                                             |
| `status`            | ENUM         | No       | `graduated` is distinct from `inactive` — graduated students completed their program vs. dropped out or left. Default `active`. |
| `created_at`        | TIMESTAMPTZ  | No       | —                                                                                                                               |
| `updated_at`        | TIMESTAMPTZ  | No       | —                                                                                                                               |

**Relationships:**

- Has many `enrollments`
- Optionally referenced by `respondents.student_id`

---

### `GET /students`

**Query params:** `status`, `search` (searches name + email), `page`, `per_page`

**Response `200`:**

```json
{
  "data": [
    {
      "id": "019687a2-0004-7000-8000-000000000001",
      "first_name": "Rina",
      "last_name": "Kusuma",
      "email": "rina@email.com",
      "phone": "082111111111",
      "date_of_birth": "2010-03-15",
      "gender": "female",
      "address": "Jl. Merdeka No. 12, Jakarta",
      "parent_name": "Dewi Kusuma",
      "parent_phone": "081222222222",
      "registration_date": "2025-01-05",
      "status": "active",
      "created_at": "2025-01-05T04:00:00Z",
      "updated_at": "2025-01-05T04:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 87,
    "total_pages": 5
  }
}
```

---

### `POST /students`

**Request:**

```json
{
  "first_name": "Rina",
  "last_name": "Kusuma",
  "email": "rina@email.com",
  "phone": "082111111111",
  "date_of_birth": "2010-03-15",
  "gender": "female",
  "address": "Jl. Merdeka No. 12, Jakarta",
  "parent_name": "Dewi Kusuma",
  "parent_phone": "081222222222",
  "registration_date": "2025-01-05"
}
```

**Response `201`:** Full student object (same shape as list item).

---

### `GET /students/:id`

**Response `200`:** Full student object. Includes `enrollments` summary array:

```json
{
  "id": "019687a2-0004-7000-8000-000000000001",
  "first_name": "Rina",
  "last_name": "Kusuma",
  "email": "rina@email.com",
  "phone": "082111111111",
  "date_of_birth": "2010-03-15",
  "gender": "female",
  "address": "Jl. Merdeka No. 12, Jakarta",
  "parent_name": "Dewi Kusuma",
  "parent_phone": "081222222222",
  "registration_date": "2025-01-05",
  "status": "active",
  "created_at": "2025-01-05T04:00:00Z",
  "updated_at": "2025-01-05T04:00:00Z",
  "enrollments": [
    {
      "id": "019687a2-0005-7000-8000-000000000001",
      "batch": {
        "id": "019687a2-0003-7000-8000-000000000001",
        "batch_name": "Matematika Dasar Batch 3",
        "batch_code": "B003",
        "status": "ongoing"
      },
      "course": {
        "id": "019687a2-0002-7000-8000-000000000001",
        "course_name": "Matematika Dasar",
        "course_code": "MTK01",
        "level": "beginner"
      },
      "status": "enrolled",
      "payment_status": "paid",
      "enrollment_date": "2025-01-10"
    }
  ]
}
```

---

### `PATCH /students/:id`

**Request** (all optional):

```json
{
  "phone": "082199999999",
  "address": "Jl. Baru No. 5, Bandung",
  "status": "graduated"
}
```

**Response `200`:** Full updated student object (without `enrollments` array).

---

## 7. Courses

### DB Table: `courses`

| Column           | Type          | Nullable | Rationale                                                                                                  |
| ---------------- | ------------- | -------- | ---------------------------------------------------------------------------------------------------------- |
| `id`             | UUID v7       | No       | PK                                                                                                         |
| `course_name`    | VARCHAR(200)  | No       | Display name.                                                                                              |
| `course_code`    | VARCHAR(20)   | No       | Short code used in batch naming and references. Globally unique. Uppercase alphanumeric.                   |
| `description`    | TEXT          | Yes      | Extended description shown in course detail view.                                                          |
| `subject`        | VARCHAR(100)  | Yes      | Subject area (e.g. "Mathematics", "English"). Useful for filtering.                                        |
| `level`          | ENUM          | Yes      | Difficulty level — drives UI badge color and filtering.                                                    |
| `session_count`  | INT           | Yes      | Maximum number of scheduled sessions per batch of this course. Enforced when adding sessions to a batch's calendar. |
| `price`          | NUMERIC(12,2) | Yes      | Course fee in IDR. Stored as fixed-point, not float.                                                       |
| `max_capacity`   | INT           | Yes      | Maximum students per batch of this course. Enforced at enrollment.                                         |
| `status`         | ENUM          | No       | `archived` courses are hidden from create-batch flows but preserved for historical data. Default `active`. |
| `created_at`     | TIMESTAMPTZ   | No       | —                                                                                                          |
| `updated_at`     | TIMESTAMPTZ   | No       | —                                                                                                          |

**Relationships:**

- Has many `batches` (one course → many batches)
- Denormalized FK on `enrollments.course_id` and `schedules.course_id`

---

### `GET /courses`

**Query params:** `status`, `level`, `search`, `page`, `per_page`

**Response `200`:**

```json
{
  "data": [
    {
      "id": "019687a2-0002-7000-8000-000000000001",
      "course_name": "Matematika Dasar",
      "course_code": "MTK01",
      "description": "Kursus matematika untuk siswa SD kelas 4-6.",
      "subject": "Mathematics",
      "level": "beginner",
      "session_count": 12,
      "price": "750000.00",
      "max_capacity": 20,
      "status": "active",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z",
      "batch_count": 3,
      "ongoing_batch_count": 1,
      "enrolled_count": 14
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 8,
    "total_pages": 1
  }
}
```

> `price` is returned as a string to avoid floating-point precision loss in JSON.  
> `batch_count` counts all batches for this course. `ongoing_batch_count` counts only `ongoing` batches. `enrolled_count` counts all non-dropped enrollments across all batches.

---

### `GET /courses/:id`

**Response `200`:** Course object with stats (same shape as a single item from the list).

```json
{
  "id": "019687a2-0002-7000-8000-000000000001",
  "course_name": "Matematika Dasar",
  "course_code": "MTK01",
  "description": "Kursus matematika untuk siswa SD kelas 4-6.",
  "subject": "Mathematics",
  "level": "beginner",
  "session_count": 12,
  "price": "750000.00",
  "max_capacity": 20,
  "status": "active",
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-01-01T00:00:00Z",
  "batch_count": 3,
  "ongoing_batch_count": 1,
  "enrolled_count": 14
}
```

**Error `404`** if not found.

---

### `POST /courses`

**Request:**

```json
{
  "course_name": "Matematika Dasar",
  "course_code": "MTK01",
  "description": "Kursus matematika untuk siswa SD kelas 4-6.",
  "subject": "Mathematics",
  "level": "beginner",
  "session_count": 12,
  "price": "750000.00",
  "max_capacity": 20
}
```

**Response `201`:** Full course object.

---

### `PATCH /courses/:id`

**Request** (all optional):

```json
{
  "course_name": "Matematika Dasar (Revisi)",
  "price": "800000.00",
  "max_capacity": 25
}
```

**Response `200`:** Full updated course object.

---

### `PATCH /courses/:id/archive`

No request body.

**Response `200`:**

```json
{
  "id": "019687a2-0002-7000-8000-000000000001",
  "status": "archived",
  "updated_at": "2025-05-18T03:00:00Z"
}
```

**Error `422`** if course has ongoing batches:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Cannot archive a course with ongoing batches"
  }
}
```

---

## 8. Batches

### DB Table: `batches`

| Column               | Type         | Nullable | Rationale                                                                                                                                      |
| -------------------- | ------------ | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`                 | UUID v7      | No       | PK                                                                                                                                             |
| `course_id`          | UUID         | No       | FK → courses. A batch belongs to exactly one course. Batch numbering is scoped per course — two courses can each have "B001".                  |
| `instructor_user_id` | UUID         | No       | FK → users (role = teacher). The default instructor for all sessions in this batch. Individual schedule slots and overrides can override this. |
| `created_by_user_id` | UUID         | No       | FK → users. Audit trail — who created the batch. Server-set from JWT claims, not from request body.                                            |
| `batch_name`         | VARCHAR(200) | No       | Human-readable name, e.g. "Matematika Dasar Batch 3".                                                                                          |
| `batch_code`         | VARCHAR(20)  | No       | Short code unique per course, e.g. "B003". Unique constraint: `(course_id, batch_code)`.                                                       |
| `start_date`         | DATE         | Yes      | When the batch starts. Does not auto-transition status — status is manually managed.                                                           |
| `end_date`           | DATE         | Yes      | When the batch ends. Informational only.                                                                                                       |
| `academic_year`      | VARCHAR(10)  | Yes      | e.g. "2025/2026". Used for filtering and grouping.                                                                                             |
| `status`             | ENUM         | No       | Explicit state machine: `upcoming → ongoing → completed`. No reversals. Default `upcoming`.                                                    |
| `created_at`         | TIMESTAMPTZ  | No       | —                                                                                                                                              |
| `updated_at`         | TIMESTAMPTZ  | No       | —                                                                                                                                              |

**Relationships:**

- Belongs to `courses`
- Belongs to `users` (instructor, creator)
- Has many `enrollments`
- Has many `schedules`

---

### `GET /batches`

**Query params:** `course_id`, `status`, `academic_year`, `page`, `per_page`

**Response `200`:**

```json
{
  "data": [
    {
      "id": "019687a2-0003-7000-8000-000000000001",
      "batch_name": "Matematika Dasar Batch 3",
      "batch_code": "B003",
      "academic_year": "2025/2026",
      "start_date": "2025-02-01",
      "end_date": "2025-05-01",
      "status": "ongoing",
      "course": {
        "id": "019687a2-0002-7000-8000-000000000001",
        "course_name": "Matematika Dasar",
        "course_code": "MTK01",
        "level": "beginner"
      },
      "instructor": {
        "id": "019687a2-0001-7000-8000-000000000001",
        "first_name": "Budi",
        "last_name": "Santoso",
        "email": "budi@tutorplace.id",
        "role": "teacher"
      },
      "created_by": {
        "id": "019687a2-0001-7000-8000-000000000099",
        "first_name": "Admin",
        "last_name": "Utama",
        "email": "admin@tutorplace.id",
        "role": "admin"
      },
      "enrolled_count": 12,
      "created_at": "2025-01-15T03:00:00Z",
      "updated_at": "2025-02-01T03:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 3,
    "total_pages": 1
  }
}
```

> `enrolled_count` is computed via `COUNT(enrollments WHERE status != 'dropped')`. Included in list and detail to power the capacity bar in the UI without a separate request.

---

### `POST /batches`

**Request:**

```json
{
  "course_id": "019687a2-0002-7000-8000-000000000001",
  "instructor_user_id": "019687a2-0001-7000-8000-000000000001",
  "batch_name": "Matematika Dasar Batch 3",
  "batch_code": "B003",
  "start_date": "2025-02-01",
  "end_date": "2025-05-01",
  "academic_year": "2025/2026"
}
```

> `created_by_user_id` is set server-side from the JWT — do not send in body.

**Response `201`:** Full batch object (same shape as list item).

---

### `GET /batches/:id`

**Response `200`:** Same as list item shape — `enrolled_count` and nested `course`, `instructor`, `created_by` always included.

---

### `PATCH /batches/:id`

**Request** (all optional):

```json
{
  "instructor_user_id": "019687a2-0001-7000-8000-000000000002",
  "batch_name": "Matematika Dasar Batch 3 (Sore)",
  "start_date": "2025-02-15"
}
```

**Response `200`:** Full updated batch object.

---

### `PATCH /batches/:id/status`

**Request:**

```json
{
  "status": "ongoing"
}
```

**Response `200`:**

```json
{
  "id": "019687a2-0003-7000-8000-000000000001",
  "status": "ongoing",
  "updated_at": "2025-05-18T03:00:00Z"
}
```

**Error `422`** on invalid transition:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid status transition: completed → ongoing"
  }
}
```

---

## 9. Enrollments

### DB Table: `enrollments`

| Column            | Type        | Nullable | Rationale                                                                                                                                                   |
| ----------------- | ----------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`              | UUID v7     | No       | PK                                                                                                                                                          |
| `student_id`      | UUID        | No       | FK → students                                                                                                                                               |
| `batch_id`        | UUID        | No       | FK → batches. Together with `student_id`, forms a unique constraint.                                                                                        |
| `course_id`       | UUID        | No       | FK → courses. Denormalized from `batch.course_id` for direct join convenience. Must always match — server sets this from batch, client does not provide it. |
| `enrollment_date` | DATE        | No       | When the student was officially enrolled. Defaults to today if not provided.                                                                                |
| `status`          | ENUM        | No       | `dropped` records are preserved for audit — not deleted. `completed` set when the batch completes. Default `enrolled`.                                      |
| `payment_status`  | ENUM        | No       | Tracks fee payment independently from enrollment status. Default `pending`.                                                                                 |
| `final_grade`     | VARCHAR(10) | Yes      | Only meaningful when `status = completed`. Free-text (e.g. "A", "85", "Lulus").                                                                             |
| `created_at`      | TIMESTAMPTZ | No       | —                                                                                                                                                           |
| `updated_at`      | TIMESTAMPTZ | No       | —                                                                                                                                                           |

**Unique constraint:** `(student_id, batch_id)`

**Relationships:**

- Belongs to `students`, `batches`, `courses`

---

### `GET /enrollments`

**Query params:** `batch_id`, `student_id`, `status`, `payment_status`, `page`, `per_page`

**Response `200`:**

```json
{
  "data": [
    {
      "id": "019687a2-0005-7000-8000-000000000001",
      "enrollment_date": "2025-01-10",
      "status": "enrolled",
      "payment_status": "paid",
      "final_grade": null,
      "student": {
        "id": "019687a2-0004-7000-8000-000000000001",
        "first_name": "Rina",
        "last_name": "Kusuma",
        "email": "rina@email.com"
      },
      "batch": {
        "id": "019687a2-0003-7000-8000-000000000001",
        "batch_name": "Matematika Dasar Batch 3",
        "batch_code": "B003",
        "status": "ongoing"
      },
      "course": {
        "id": "019687a2-0002-7000-8000-000000000001",
        "course_name": "Matematika Dasar",
        "course_code": "MTK01",
        "level": "beginner"
      },
      "created_at": "2025-01-10T04:00:00Z",
      "updated_at": "2025-01-10T04:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 12,
    "total_pages": 1
  }
}
```

---

### `POST /enrollments`

**Request:**

```json
{
  "student_id": "019687a2-0004-7000-8000-000000000001",
  "batch_id": "019687a2-0003-7000-8000-000000000001",
  "enrollment_date": "2025-01-10"
}
```

> `course_id` is not accepted from client — server derives it from `batch.course_id`.

**Response `201`:** Full enrollment object.

**Error `409`** on duplicate:

```json
{
  "error": {
    "code": "CONFLICT",
    "message": "Student is already enrolled in this batch"
  }
}
```

**Error `422`** on capacity full:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Batch has reached maximum capacity"
  }
}
```

---

### `PATCH /enrollments/:id`

**Request** (all optional):

```json
{
  "status": "completed",
  "payment_status": "paid",
  "final_grade": "A"
}
```

**Response `200`:** Full updated enrollment object.

---

## 10. Schedules

### DB Table: `schedules`

| Column               | Type         | Nullable | Rationale                                                                                                                                                                     |
| -------------------- | ------------ | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`                 | UUID v7      | No       | PK                                                                                                                                                                            |
| `batch_id`           | UUID         | No       | FK → batches. A schedule slot belongs to one batch.                                                                                                                           |
| `course_id`          | UUID         | No       | FK → courses. Denormalized from batch — server sets, client does not send.                                                                                                    |
| `instructor_user_id` | UUID         | Yes      | FK → users. Slot-level instructor override. If `null`, the batch's default `instructor_user_id` applies. This enables e.g. a guest lecturer for one subject within the batch. |
| `day_of_week`        | ENUM         | No       | The recurring day. Interpreted in WIB (UTC+7).                                                                                                                                |
| `start_time`         | TIME         | No       | Session start, 24h, WIB. Stored as `TIME` (no timezone).                                                                                                                      |
| `end_time`           | TIME         | No       | Session end, 24h, WIB. Must be after `start_time`.                                                                                                                            |
| `room`               | VARCHAR(100) | Yes      | Physical or virtual room identifier.                                                                                                                                          |
| `recurrence`         | ENUM         | No       | `weekly` = repeats every week on `day_of_week`. `one-time` = occurs once on the date defined by `effective_from`.                                                             |
| `effective_from`     | DATE         | No       | Date from which this slot is active. For `one-time`, this is the single session date.                                                                                         |
| `effective_until`    | DATE         | Yes      | Date until which this slot is active. `null` means open-ended (runs until batch ends). For `one-time`, matches `effective_from`.                                              |
| `created_at`         | TIMESTAMPTZ  | No       | —                                                                                                                                                                             |
| `updated_at`         | TIMESTAMPTZ  | No       | —                                                                                                                                                                             |

**Relationships:**

- Belongs to `batches`
- Has many `schedule_overrides`

---

### `GET /batches/:batch_id/schedules`

**Response `200`:**

```json
{
  "data": [
    {
      "id": "019687a2-0006-7000-8000-000000000001",
      "day_of_week": "monday",
      "start_time": "09:00:00",
      "end_time": "10:30:00",
      "room": "Ruang A1",
      "recurrence": "weekly",
      "effective_from": "2025-02-01",
      "effective_until": null,
      "instructor": null,
      "effective_instructor": {
        "id": "019687a2-0001-7000-8000-000000000001",
        "first_name": "Budi",
        "last_name": "Santoso",
        "email": "budi@tutorplace.id",
        "role": "teacher"
      },
      "instructor_source": "batch",
      "batch": {
        "id": "019687a2-0003-7000-8000-000000000001",
        "batch_name": "Matematika Dasar Batch 3",
        "batch_code": "B003",
        "status": "ongoing"
      },
      "created_at": "2025-01-15T03:00:00Z",
      "updated_at": "2025-01-15T03:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 3,
    "total_pages": 1
  }
}
```

> `instructor` — the slot-level override (nullable). `null` means this slot inherits from the batch.  
> `effective_instructor` — always resolved and returned; the instructor that will actually teach this slot (3-level fallback applied server-side).  
> `instructor_source` — one of `"slot"` | `"batch"` | `"override"` — tells the UI where the effective instructor came from.

---

### `POST /batches/:batch_id/schedules`

**Request:**

```json
{
  "day_of_week": "monday",
  "start_time": "09:00:00",
  "end_time": "10:30:00",
  "room": "Ruang A1",
  "recurrence": "weekly",
  "effective_from": "2025-02-01",
  "effective_until": null,
  "instructor_user_id": null
}
```

> `instructor_user_id: null` means inherit from batch. Omitting the field has the same effect.

**Response `201`:** Full schedule object.

---

### `PATCH /schedules/:id`

**Request** (all optional):

```json
{
  "room": "Ruang B2",
  "end_time": "11:00:00"
}
```

**Response `200`:** Full updated schedule object.

---

## 11. Schedule Overrides

### DB Table: `schedule_overrides`

| Column                   | Type         | Nullable | Rationale                                                                                                                                                              |
| ------------------------ | ------------ | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`                     | UUID v7      | No       | PK                                                                                                                                                                     |
| `schedule_id`            | UUID         | No       | FK → schedules. Which recurring slot is being overridden.                                                                                                              |
| `original_date`          | DATE         | No       | The specific calendar date of the session being affected. Combined with `schedule_id`, must be unique — only one override per session per slot.                        |
| `override_type`          | ENUM         | No       | Determines which other fields are meaningful: `reschedule` changes time/date/room; `cancellation` voids the session; `instructor_change` replaces only the instructor. |
| `new_date`               | DATE         | Yes      | Required when `override_type = reschedule`. The rescheduled date.                                                                                                      |
| `new_start_time`         | TIME         | Yes      | New start time. Only set for `reschedule`.                                                                                                                             |
| `new_end_time`           | TIME         | Yes      | New end time. Only set for `reschedule`.                                                                                                                               |
| `new_room`               | VARCHAR(100) | Yes      | New room. Can be set for `reschedule` or standalone room change.                                                                                                       |
| `new_instructor_user_id` | UUID         | Yes      | FK → users. Required for `instructor_change`. Optional for `reschedule`. Null for `cancellation`.                                                                      |
| `reason`                 | TEXT         | Yes      | Free-text explanation. Shown in override list for auditing.                                                                                                            |
| `created_by_user_id`     | UUID         | No       | FK → users. Server-set from JWT — who logged this override.                                                                                                            |
| `created_at`             | TIMESTAMPTZ  | No       | —                                                                                                                                                                      |

**Unique constraint:** `(schedule_id, original_date)`

---

### `GET /schedules/:schedule_id/overrides`

**Response `200`:**

```json
{
  "data": [
    {
      "id": "019687a2-0007-7000-8000-000000000001",
      "schedule_id": "019687a2-0006-7000-8000-000000000001",
      "original_date": "2025-03-10",
      "override_type": "instructor_change",
      "new_date": null,
      "new_start_time": null,
      "new_end_time": null,
      "new_room": null,
      "new_instructor": {
        "id": "019687a2-0001-7000-8000-000000000002",
        "first_name": "Sari",
        "last_name": "Indah",
        "email": "sari@tutorplace.id",
        "role": "teacher"
      },
      "reason": "Budi sedang sakit",
      "created_by": {
        "id": "019687a2-0001-7000-8000-000000000099",
        "first_name": "Admin",
        "last_name": "Utama",
        "email": "admin@tutorplace.id",
        "role": "admin"
      },
      "created_at": "2025-03-09T05:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

---

### `POST /schedules/:schedule_id/overrides`

**Request — `instructor_change`:**

```json
{
  "original_date": "2025-03-10",
  "override_type": "instructor_change",
  "new_instructor_user_id": "019687a2-0001-7000-8000-000000000002",
  "reason": "Budi sedang sakit"
}
```

**Request — `reschedule`:**

```json
{
  "original_date": "2025-03-10",
  "override_type": "reschedule",
  "new_date": "2025-03-12",
  "new_start_time": "13:00:00",
  "new_end_time": "14:30:00",
  "new_room": "Ruang C3",
  "reason": "Bentrok dengan acara sekolah"
}
```

**Request — `cancellation`:**

```json
{
  "original_date": "2025-03-17",
  "override_type": "cancellation",
  "reason": "Libur nasional"
}
```

**Response `201`:** Full override object (same shape as list item).

---

## 12. Events

### DB Table: `events`

| Column               | Type         | Nullable | Rationale                                                                                                                 |
| -------------------- | ------------ | -------- | ------------------------------------------------------------------------------------------------------------------------- |
| `id`                 | UUID v7      | No       | PK                                                                                                                        |
| `title`              | VARCHAR(200) | No       | —                                                                                                                         |
| `description`        | TEXT         | Yes      | Extended detail, shown in event detail view.                                                                              |
| `event_type`         | ENUM         | No       | Drives UI color-coding and filtering.                                                                                     |
| `start_datetime`     | TIMESTAMPTZ  | No       | UTC. Displayed in WIB on frontend.                                                                                        |
| `end_datetime`       | TIMESTAMPTZ  | No       | UTC. Must be after `start_datetime`.                                                                                      |
| `location`           | VARCHAR(200) | Yes      | Physical address or virtual meeting link.                                                                                 |
| `audience`           | ENUM         | No       | Who the event targets. `specific_batch` is a flag — actual batch linking via future `event_batches` join table if needed. |
| `created_by_user_id` | UUID         | No       | FK → users. Server-set from JWT.                                                                                          |
| `created_at`         | TIMESTAMPTZ  | No       | —                                                                                                                         |
| `updated_at`         | TIMESTAMPTZ  | No       | —                                                                                                                         |

---

### `GET /events`

**Query params:** `event_type`, `audience`, `from` (datetime), `until` (datetime), `page`, `per_page`

**Response `200`:**

```json
{
  "data": [
    {
      "id": "019687a2-0008-7000-8000-000000000001",
      "title": "Ujian Tengah Semester",
      "description": "UTS untuk semua kelas Matematika.",
      "event_type": "exam",
      "start_datetime": "2025-03-20T01:00:00Z",
      "end_datetime": "2025-03-20T04:00:00Z",
      "location": "Ruang Utama",
      "audience": "students",
      "created_by": {
        "id": "019687a2-0001-7000-8000-000000000099",
        "first_name": "Admin",
        "last_name": "Utama",
        "email": "admin@tutorplace.id",
        "role": "admin"
      },
      "created_at": "2025-03-01T03:00:00Z",
      "updated_at": "2025-03-01T03:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 4,
    "total_pages": 1
  }
}
```

---

### `POST /events`

**Request:**

```json
{
  "title": "Ujian Tengah Semester",
  "description": "UTS untuk semua kelas Matematika.",
  "event_type": "exam",
  "start_datetime": "2025-03-20T01:00:00Z",
  "end_datetime": "2025-03-20T04:00:00Z",
  "location": "Ruang Utama",
  "audience": "students"
}
```

**Response `201`:** Full event object.

---

### `PATCH /events/:id`

**Request** (all optional):

```json
{
  "location": "Ruang B dan C",
  "end_datetime": "2025-03-20T05:00:00Z"
}
```

**Response `200`:** Full updated event object.

---

## 13. Forms

### DB Table: `forms`

| Column               | Type         | Nullable | Rationale                                                                                                                                    |
| -------------------- | ------------ | -------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`                 | UUID v7      | No       | PK                                                                                                                                           |
| `title`              | VARCHAR(200) | No       | —                                                                                                                                            |
| `description`        | TEXT         | Yes      | Shown at the top of the public form page.                                                                                                    |
| `status`             | ENUM         | No       | Lifecycle state: `draft → published → closed → deleted`. Default `draft`.                                                                    |
| `allow_anonymous`    | BOOLEAN      | No       | If `false`, respondent must provide name + email (and system links to student if matched). If `true`, no identity required. Default `false`. |
| `created_by_user_id` | UUID         | No       | FK → users. Server-set from JWT.                                                                                                             |
| `created_at`         | TIMESTAMPTZ  | No       | —                                                                                                                                            |
| `updated_at`         | TIMESTAMPTZ  | No       | —                                                                                                                                            |
| `published_at`       | TIMESTAMPTZ  | Yes      | Set when status transitions to `published`. Used to show publish date in UI.                                                                 |
| `closed_at`          | TIMESTAMPTZ  | Yes      | Set when status transitions to `closed`.                                                                                                     |
| `deleted_at`         | TIMESTAMPTZ  | Yes      | Soft delete. Rows with `deleted_at IS NOT NULL` excluded from all listings.                                                                  |

**Relationships:**

- Has many `form_questions`
- Has many `form_responses`

---

### `GET /forms`

**Query params:** `status`, `page`, `per_page`. Deleted forms always excluded.

**Response `200`:**

```json
{
  "data": [
    {
      "id": "019687a2-0009-7000-8000-000000000001",
      "title": "Survei Kepuasan Siswa",
      "description": "Bantu kami meningkatkan kualitas pengajaran.",
      "status": "published",
      "allow_anonymous": false,
      "question_count": 5,
      "response_count": 23,
      "created_by": {
        "id": "019687a2-0001-7000-8000-000000000099",
        "first_name": "Admin",
        "last_name": "Utama",
        "email": "admin@tutorplace.id",
        "role": "admin"
      },
      "created_at": "2025-04-01T03:00:00Z",
      "updated_at": "2025-04-05T03:00:00Z",
      "published_at": "2025-04-05T03:00:00Z",
      "closed_at": null
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 6,
    "total_pages": 1
  }
}
```

> `question_count` and `response_count` are computed aggregates — included in list to avoid extra requests on the forms index page.

---

### `GET /forms/:id`

**Response `200`:** Full form object with `questions` array embedded.

```json
{
  "id": "019687a2-0009-7000-8000-000000000001",
  "title": "Survei Kepuasan Siswa",
  "description": "Bantu kami meningkatkan kualitas pengajaran.",
  "status": "published",
  "allow_anonymous": false,
  "public_url": "https://app.tutorplace.id/f/019687a2-0009-7000-8000-000000000001",
  "question_count": 5,
  "response_count": 23,
  "created_by": {
    "id": "019687a2-0001-7000-8000-000000000099",
    "first_name": "Admin",
    "last_name": "Utama",
    "email": "admin@tutorplace.id",
    "role": "admin"
  },
  "created_at": "2025-04-01T03:00:00Z",
  "updated_at": "2025-04-05T03:00:00Z",
  "published_at": "2025-04-05T03:00:00Z",
  "closed_at": null,
  "questions": [
    {
      "id": "019687a2-0010-7000-8000-000000000001",
      "question_text": "Bagaimana penilaian Anda terhadap pengajaran instruktur?",
      "question_type": "rating",
      "is_required": true,
      "order_index": 1,
      "options": null
    },
    {
      "id": "019687a2-0010-7000-8000-000000000002",
      "question_text": "Materi apa yang paling membantu?",
      "question_type": "multiple_choice",
      "is_required": false,
      "order_index": 2,
      "options": ["Aljabar", "Geometri", "Statistika", "Kalkulus"]
    }
  ]
}
```

---

### `POST /forms`

**Request:**

```json
{
  "title": "Survei Kepuasan Siswa",
  "description": "Bantu kami meningkatkan kualitas pengajaran.",
  "allow_anonymous": false
}
```

**Response `201`:** Full form object (no questions yet).

---

### `PATCH /forms/:id`

Only allowed when `status = draft`.

**Request** (all optional):

```json
{
  "title": "Survei Kepuasan Siswa (Revisi)",
  "allow_anonymous": true
}
```

**Response `200`:** Full updated form object.

---

### `POST /forms/:id/publish`

No request body. Idempotent — returns `200` if already published.

**Response `200`:**

```json
{
  "id": "019687a2-0009-7000-8000-000000000001",
  "status": "published",
  "published_at": "2025-05-18T03:00:00Z",
  "public_url": "https://app.tutorplace.id/f/019687a2-0009-7000-8000-000000000001"
}
```

---

### `POST /forms/:id/close`

No request body. Idempotent.

**Response `200`:**

```json
{
  "id": "019687a2-0009-7000-8000-000000000001",
  "status": "closed",
  "closed_at": "2025-05-18T03:00:00Z"
}
```

---

### `DELETE /forms/:id`

Soft delete — sets `deleted_at` and `status = deleted`. No body.

**Response `200`:**

```json
{
  "id": "019687a2-0009-7000-8000-000000000001",
  "status": "deleted",
  "deleted_at": "2025-05-18T03:00:00Z"
}
```

---

## 14. Form Questions

### DB Table: `form_questions`

| Column          | Type    | Nullable | Rationale                                                                                                                                    |
| --------------- | ------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`            | UUID v7 | No       | PK                                                                                                                                           |
| `form_id`       | UUID    | No       | FK → forms.                                                                                                                                  |
| `question_text` | TEXT    | No       | The question as shown to the respondent.                                                                                                     |
| `question_type` | ENUM    | No       | Drives the input rendered on the public form page and the aggregate view in the response viewer.                                             |
| `is_required`   | BOOLEAN | No       | If `true`, respondent cannot submit without answering. Default `false`.                                                                      |
| `order_index`   | INT     | No       | Display order. 1-based. Managed by the frontend drag-to-reorder — sent on PATCH.                                                             |
| `options`       | JSONB   | Yes      | Array of option strings. Only present for `multiple_choice` and `checkbox`. Null for all other types. Stored as JSONB for flexible querying. |

**Immutability rule:** Questions cannot be created, edited, or deleted once their parent form is `published`.

---

### `POST /forms/:id/questions`

**Request — `multiple_choice`:**

```json
{
  "question_text": "Materi apa yang paling membantu?",
  "question_type": "multiple_choice",
  "is_required": false,
  "order_index": 2,
  "options": ["Aljabar", "Geometri", "Statistika", "Kalkulus"]
}
```

**Request — `text`:**

```json
{
  "question_text": "Saran atau masukan Anda?",
  "question_type": "text",
  "is_required": false,
  "order_index": 5,
  "options": null
}
```

**Request — `rating`:**

```json
{
  "question_text": "Berikan penilaian untuk instruktur (1-5)",
  "question_type": "rating",
  "is_required": true,
  "order_index": 1,
  "options": null
}
```

**Response `201`:** Full question object.

```json
{
  "id": "019687a2-0010-7000-8000-000000000001",
  "form_id": "019687a2-0009-7000-8000-000000000001",
  "question_text": "Berikan penilaian untuk instruktur (1-5)",
  "question_type": "rating",
  "is_required": true,
  "order_index": 1,
  "options": null
}
```

---

### `PATCH /form-questions/:id`

**Request** (all optional, draft only):

```json
{
  "question_text": "Berikan nilai untuk instruktur (1-10)",
  "is_required": true,
  "order_index": 3,
  "options": null
}
```

**Response `200`:** Full updated question object.

---

### `DELETE /form-questions/:id`

Hard delete. Only allowed on draft forms.

**Response `200`:**

```json
{
  "id": "019687a2-0010-7000-8000-000000000001",
  "deleted": true
}
```

---

## 15. Form Responses

### DB Tables: `respondents`, `form_responses`, `form_answers`

**`respondents`**

| Column       | Type         | Nullable | Rationale                                                                                                                 |
| ------------ | ------------ | -------- | ------------------------------------------------------------------------------------------------------------------------- |
| `id`         | UUID v7      | No       | PK                                                                                                                        |
| `student_id` | UUID         | Yes      | FK → students. If the respondent is a known student, linked here. Null for non-students or anonymous.                     |
| `name`       | VARCHAR(200) | Yes      | Required when `allow_anonymous = false`.                                                                                  |
| `email`      | VARCHAR(255) | Yes      | Required when `allow_anonymous = false`. Used as upsert key — same email across multiple forms reuses the respondent row. |
| `created_at` | TIMESTAMPTZ  | No       | —                                                                                                                         |

**`form_responses`**

| Column          | Type        | Nullable | Rationale                      |
| --------------- | ----------- | -------- | ------------------------------ |
| `id`            | UUID v7     | No       | PK                             |
| `form_id`       | UUID        | No       | FK → forms                     |
| `respondent_id` | UUID        | No       | FK → respondents               |
| `submitted_at`  | TIMESTAMPTZ | No       | Server-set at submission time. |

**Partial unique index:** `(form_id, respondent_id) WHERE form.allow_anonymous = false` — prevents duplicate submissions from identified respondents.

**`form_answers`**

| Column         | Type    | Nullable | Rationale                                                                                                                      |
| -------------- | ------- | -------- | ------------------------------------------------------------------------------------------------------------------------------ |
| `id`           | UUID v7 | No       | PK                                                                                                                             |
| `response_id`  | UUID    | No       | FK → form_responses                                                                                                            |
| `question_id`  | UUID    | No       | FK → form_questions. Validated that `question.form_id = response.form_id`.                                                     |
| `answer_text`  | TEXT    | Yes      | Used for `text` and `date` question types. For `date`, stored as ISO 8601 date string.                                         |
| `answer_value` | JSONB   | Yes      | Used for `multiple_choice` (single string), `checkbox` (array of strings), and `rating` (integer). Null for `text` and `date`. |

---

### `POST /forms/:id/responses` _(public, no auth required)_

**Request — anonymous form (`allow_anonymous = true`):**

```json
{
  "answers": [
    {
      "question_id": "019687a2-0010-7000-8000-000000000001",
      "answer_text": null,
      "answer_value": 4
    },
    {
      "question_id": "019687a2-0010-7000-8000-000000000002",
      "answer_text": null,
      "answer_value": "Aljabar"
    },
    {
      "question_id": "019687a2-0010-7000-8000-000000000003",
      "answer_text": "Terima kasih, sangat membantu!",
      "answer_value": null
    }
  ]
}
```

**Request — identified form (`allow_anonymous = false`):**

```json
{
  "respondent": {
    "name": "Rina Kusuma",
    "email": "rina@email.com"
  },
  "answers": [
    {
      "question_id": "019687a2-0010-7000-8000-000000000001",
      "answer_text": null,
      "answer_value": 5
    }
  ]
}
```

> **`answer_value` shape by question type:**
>
> - `rating` → integer: `4`
> - `multiple_choice` → string: `"Aljabar"`
> - `checkbox` → array of strings: `["Aljabar", "Geometri"]`
> - `text` → use `answer_text`, leave `answer_value: null`
> - `date` → use `answer_text` as `"2025-05-18"`, leave `answer_value: null`

**Response `201`:**

```json
{
  "response_id": "019687a2-0011-7000-8000-000000000001",
  "submitted_at": "2025-05-18T03:00:00Z"
}
```

---

### `GET /forms/:id/responses` _(admin/staff only)_

**Response `200`:**

```json
{
  "data": [
    {
      "id": "019687a2-0011-7000-8000-000000000001",
      "submitted_at": "2025-05-18T03:00:00Z",
      "respondent": {
        "id": "019687a2-0012-7000-8000-000000000001",
        "name": "Rina Kusuma",
        "email": "rina@email.com",
        "student": {
          "id": "019687a2-0004-7000-8000-000000000001",
          "first_name": "Rina",
          "last_name": "Kusuma"
        }
      },
      "answers": [
        {
          "question_id": "019687a2-0010-7000-8000-000000000001",
          "question_text": "Berikan penilaian untuk instruktur (1-5)",
          "question_type": "rating",
          "answer_text": null,
          "answer_value": 5
        },
        {
          "question_id": "019687a2-0010-7000-8000-000000000002",
          "question_text": "Materi apa yang paling membantu?",
          "question_type": "multiple_choice",
          "answer_text": null,
          "answer_value": "Aljabar"
        }
      ]
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 23,
    "total_pages": 2
  }
}
```

---

### `GET /forms/:id/responses/export`

Returns a CSV file download. No pagination — exports all responses.

**Response headers:**

```
Content-Type: text/csv; charset=utf-8
Content-Disposition: attachment; filename="survei-kepuasan-siswa-responses.csv"
```

---

## 16. Auth

### DB Table: `refresh_tokens`

| Column       | Type        | Nullable | Rationale                                                         |
| ------------ | ----------- | -------- | ----------------------------------------------------------------- |
| `id`         | UUID v7     | No       | PK                                                                |
| `user_id`    | UUID        | No       | FK → users                                                        |
| `token_hash` | VARCHAR(64) | No       | SHA-256 of the raw refresh token. Raw token never stored. Unique. |
| `expires_at` | TIMESTAMPTZ | No       | Server-set. 7 days from issuance.                                 |
| `created_at` | TIMESTAMPTZ | No       | —                                                                 |

---

### `POST /auth/login`

**Request:**

```json
{
  "email": "budi@tutorplace.id",
  "password": "rahasia123"
}
```

**Response `200`:**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "a3f8c2e1d4b7...",
  "expires_in": 900,
  "user": {
    "id": "019687a2-0001-7000-8000-000000000001",
    "first_name": "Budi",
    "last_name": "Santoso",
    "email": "budi@tutorplace.id",
    "role": "teacher",
    "status": "active"
  }
}
```

> `expires_in` — access token TTL in seconds (900 = 15 minutes).  
> `refresh_token` — raw token; client stores this securely. Server stores only its SHA-256 hash.

---

### `POST /auth/refresh`

**Request:**

```json
{
  "refresh_token": "a3f8c2e1d4b7..."
}
```

**Response `200`:**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "z9q1p0r5t2s8...",
  "expires_in": 900
}
```

> Refresh tokens are **rotated** on every use — the old token is deleted, a new one issued. Reusing an already-rotated token returns `401`.

---

### `POST /auth/logout`

**Request:**

```json
{
  "refresh_token": "a3f8c2e1d4b7..."
}
```

**Response `200`:**

```json
{
  "message": "Logged out successfully"
}
```

---

### `POST /auth/password-reset/request`

**Request:**

```json
{
  "email": "budi@tutorplace.id"
}
```

**Response `200`:**

```json
{
  "message": "If that email exists, a reset link has been sent"
}
```

> Always returns the same response regardless of whether the email exists — prevents email enumeration.

---

### `POST /auth/password-reset/confirm`

**Request:**

```json
{
  "token": "reset-token-from-email-link",
  "password": "passwordbaru123"
}
```

**Response `200`:**

```json
{
  "message": "Password updated successfully"
}
```

**Error `422`** if token expired or invalid:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Reset token is invalid or has expired"
  }
}
```

---

## Appendix: Field Naming Cheatsheet

Quick reference for fields that appear across multiple entities.

| Field                           | Always means                                                             |
| ------------------------------- | ------------------------------------------------------------------------ |
| `id`                            | UUID v7 primary key of the resource itself                               |
| `created_at`                    | UTC datetime set by server at INSERT — never client-provided             |
| `updated_at`                    | UTC datetime updated by server at every UPDATE                           |
| `status`                        | Lifecycle state — always a string enum, never an integer                 |
| `created_by`                    | Nested `UserSummary` of who created the record — server-derived from JWT |
| `*_user_id` fields in requests  | UUID string referencing a user                                           |
| `*` nested objects in responses | Always expanded — never return bare IDs in response bodies               |
| `deleted_at`                    | Soft delete timestamp; `null` means not deleted                          |
| `answer_value`                  | JSONB — shape varies by `question_type`; see Section 15                  |
| `instructor_source`             | One of `"slot"` \| `"batch"` \| `"override"` — resolved server-side      |
