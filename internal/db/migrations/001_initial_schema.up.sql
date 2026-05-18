-- users: system users (admin, teacher, staff)
CREATE TABLE users (
    id                UUID         PRIMARY KEY,
    first_name        VARCHAR(100) NOT NULL,
    last_name         VARCHAR(100) NOT NULL,
    email             VARCHAR(255) NOT NULL UNIQUE,
    password_hash     VARCHAR(255) NOT NULL,
    role              VARCHAR(20)  NOT NULL CHECK (role IN ('admin', 'teacher', 'staff')),
    phone             VARCHAR(20),
    status            VARCHAR(20)  NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    failed_attempts   INT          NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- refresh_tokens: SHA-256 hashes only — raw tokens never stored
CREATE TABLE refresh_tokens (
    id          UUID        PRIMARY KEY,
    user_id     UUID        NOT NULL REFERENCES users(id),
    token_hash  VARCHAR(64) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id    ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- password_reset_tokens: short-lived; same hash-only pattern as refresh tokens
CREATE TABLE password_reset_tokens (
    id          UUID        PRIMARY KEY,
    user_id     UUID        NOT NULL REFERENCES users(id),
    token_hash  VARCHAR(64) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_reset_tokens_token_hash ON password_reset_tokens(token_hash);
CREATE INDEX idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at);

-- students
CREATE TABLE students (
    id                UUID         PRIMARY KEY,
    first_name        VARCHAR(100) NOT NULL,
    last_name         VARCHAR(100) NOT NULL,
    email             VARCHAR(255) UNIQUE,
    phone             VARCHAR(20),
    date_of_birth     DATE,
    gender            VARCHAR(10)  CHECK (gender IN ('male', 'female', 'other')),
    address           TEXT,
    parent_name       VARCHAR(200),
    parent_phone      VARCHAR(20),
    registration_date DATE         NOT NULL DEFAULT CURRENT_DATE,
    status            VARCHAR(20)  NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'graduated')),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_students_status ON students(status);
CREATE INDEX idx_students_email  ON students(email);

-- courses
CREATE TABLE courses (
    id             UUID           PRIMARY KEY,
    course_name    VARCHAR(200)   NOT NULL,
    course_code    VARCHAR(20)    NOT NULL UNIQUE,
    description    TEXT,
    subject        VARCHAR(100),
    level          VARCHAR(20)    CHECK (level IN ('beginner', 'intermediate', 'advanced')),
    duration_weeks INT,
    price          NUMERIC(12, 2),
    max_capacity   INT,
    status         VARCHAR(20)    NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    created_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_courses_status ON courses(status);

-- batches: scoped to one course; batch_code unique per course
CREATE TABLE batches (
    id                   UUID         PRIMARY KEY,
    course_id            UUID         NOT NULL REFERENCES courses(id),
    instructor_user_id   UUID         NOT NULL REFERENCES users(id),
    created_by_user_id   UUID         NOT NULL REFERENCES users(id),
    batch_name           VARCHAR(200) NOT NULL,
    batch_code           VARCHAR(20)  NOT NULL,
    start_date           DATE,
    end_date             DATE,
    academic_year        VARCHAR(10),
    status               VARCHAR(20)  NOT NULL DEFAULT 'upcoming' CHECK (status IN ('upcoming', 'ongoing', 'completed')),
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (course_id, batch_code)
);

CREATE INDEX idx_batches_course_id           ON batches(course_id);
CREATE INDEX idx_batches_status              ON batches(status);
CREATE INDEX idx_batches_instructor_user_id  ON batches(instructor_user_id);
CREATE INDEX idx_batches_academic_year       ON batches(academic_year);

-- enrollments: student ↔ batch M:N; course_id denormalized from batch
CREATE TABLE enrollments (
    id              UUID        PRIMARY KEY,
    student_id      UUID        NOT NULL REFERENCES students(id),
    batch_id        UUID        NOT NULL REFERENCES batches(id),
    course_id       UUID        NOT NULL REFERENCES courses(id),
    enrollment_date DATE        NOT NULL DEFAULT CURRENT_DATE,
    status          VARCHAR(20) NOT NULL DEFAULT 'enrolled' CHECK (status IN ('enrolled', 'dropped', 'completed')),
    payment_status  VARCHAR(20) NOT NULL DEFAULT 'pending'  CHECK (payment_status IN ('pending', 'partial', 'paid')),
    final_grade     VARCHAR(10),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, batch_id)
);

CREATE INDEX idx_enrollments_batch_id_status ON enrollments(batch_id, status);
CREATE INDEX idx_enrollments_student_id      ON enrollments(student_id);
CREATE INDEX idx_enrollments_course_id       ON enrollments(course_id);
CREATE INDEX idx_enrollments_status          ON enrollments(status);
CREATE INDEX idx_enrollments_payment_status  ON enrollments(payment_status);

-- schedules: recurring or one-time session slots per batch
CREATE TABLE schedules (
    id                   UUID        PRIMARY KEY,
    batch_id             UUID        NOT NULL REFERENCES batches(id),
    course_id            UUID        NOT NULL REFERENCES courses(id),
    instructor_user_id   UUID        REFERENCES users(id),
    day_of_week          VARCHAR(10) NOT NULL CHECK (day_of_week IN ('monday','tuesday','wednesday','thursday','friday','saturday','sunday')),
    start_time           TIME        NOT NULL,
    end_time             TIME        NOT NULL,
    room                 VARCHAR(100),
    recurrence           VARCHAR(10) NOT NULL CHECK (recurrence IN ('weekly', 'one-time')),
    effective_from       DATE        NOT NULL,
    effective_until      DATE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_schedules_batch_id  ON schedules(batch_id);
CREATE INDEX idx_schedules_course_id ON schedules(course_id);

-- schedule_overrides: one-off exceptions to recurring slots
CREATE TABLE schedule_overrides (
    id                       UUID        PRIMARY KEY,
    schedule_id              UUID        NOT NULL REFERENCES schedules(id),
    original_date            DATE        NOT NULL,
    override_type            VARCHAR(20) NOT NULL CHECK (override_type IN ('reschedule', 'cancellation', 'instructor_change')),
    new_date                 DATE,
    new_start_time           TIME,
    new_end_time             TIME,
    new_room                 VARCHAR(100),
    new_instructor_user_id   UUID        REFERENCES users(id),
    reason                   TEXT,
    created_by_user_id       UUID        NOT NULL REFERENCES users(id),
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (schedule_id, original_date)
);

-- events: institution-wide calendar
CREATE TABLE events (
    id                   UUID         PRIMARY KEY,
    title                VARCHAR(200) NOT NULL,
    description          TEXT,
    event_type           VARCHAR(20)  NOT NULL CHECK (event_type IN ('workshop', 'exam', 'holiday', 'meeting')),
    start_datetime       TIMESTAMPTZ  NOT NULL,
    end_datetime         TIMESTAMPTZ  NOT NULL,
    location             VARCHAR(200),
    audience             VARCHAR(20)  NOT NULL CHECK (audience IN ('all', 'students', 'staff', 'specific_batch')),
    created_by_user_id   UUID         NOT NULL REFERENCES users(id),
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_start_datetime ON events(start_datetime);
CREATE INDEX idx_events_end_datetime   ON events(end_datetime);
CREATE INDEX idx_events_audience       ON events(audience);
CREATE INDEX idx_events_event_type     ON events(event_type);

-- forms: draft → published → closed → deleted (soft)
CREATE TABLE forms (
    id                   UUID         PRIMARY KEY,
    title                VARCHAR(200) NOT NULL,
    description          TEXT,
    status               VARCHAR(20)  NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'closed', 'deleted')),
    allow_anonymous      BOOLEAN      NOT NULL DEFAULT false,
    created_by_user_id   UUID         NOT NULL REFERENCES users(id),
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    published_at         TIMESTAMPTZ,
    closed_at            TIMESTAMPTZ,
    deleted_at           TIMESTAMPTZ
);

CREATE INDEX idx_forms_status              ON forms(status);
CREATE INDEX idx_forms_deleted_at          ON forms(deleted_at);
CREATE INDEX idx_forms_created_by_user_id  ON forms(created_by_user_id);

-- form_questions: immutable once form is published
CREATE TABLE form_questions (
    id             UUID        PRIMARY KEY,
    form_id        UUID        NOT NULL REFERENCES forms(id),
    question_text  TEXT        NOT NULL,
    question_type  VARCHAR(20) NOT NULL CHECK (question_type IN ('text', 'multiple_choice', 'checkbox', 'rating', 'date')),
    is_required    BOOLEAN     NOT NULL DEFAULT false,
    order_index    INT         NOT NULL,
    options        JSONB
);

CREATE INDEX idx_form_questions_form_id_order ON form_questions(form_id, order_index);

-- respondents: decoupled from students to allow non-student respondents
CREATE TABLE respondents (
    id          UUID         PRIMARY KEY,
    student_id  UUID         REFERENCES students(id),
    name        VARCHAR(200),
    email       VARCHAR(255),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_respondents_email ON respondents(email);

-- form_responses: one submission per respondent per identified form
CREATE TABLE form_responses (
    id             UUID        PRIMARY KEY,
    form_id        UUID        NOT NULL REFERENCES forms(id),
    respondent_id  UUID        NOT NULL REFERENCES respondents(id),
    submitted_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_form_responses_form_id ON form_responses(form_id);

-- Prevents duplicate submissions from identified respondents
CREATE UNIQUE INDEX idx_form_responses_no_dup ON form_responses(form_id, respondent_id);

-- form_answers: one row per question per response
CREATE TABLE form_answers (
    id            UUID  PRIMARY KEY,
    response_id   UUID  NOT NULL REFERENCES form_responses(id),
    question_id   UUID  NOT NULL REFERENCES form_questions(id),
    answer_text   TEXT,
    answer_value  JSONB
);

CREATE INDEX idx_form_answers_response_id ON form_answers(response_id);
CREATE INDEX idx_form_answers_question_id ON form_answers(question_id);
