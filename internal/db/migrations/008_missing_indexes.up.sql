-- users
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_role       ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_status     ON users(status);

-- students
CREATE INDEX IF NOT EXISTS idx_students_created_at ON students(created_at DESC);

-- courses
CREATE INDEX IF NOT EXISTS idx_courses_created_at ON courses(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_courses_level      ON courses(level);

-- batches
CREATE INDEX IF NOT EXISTS idx_batches_created_at            ON batches(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_batches_status_created_at     ON batches(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_batches_course_id_created_at  ON batches(course_id, created_at DESC);

-- enrollments (for courses_with_stats view)
CREATE INDEX IF NOT EXISTS idx_enrollments_course_id_status  ON enrollments(course_id, status);

-- trigram indexes for ILIKE search
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_batches_batch_name_trgm  ON batches  USING GIN (batch_name  gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_batches_batch_code_trgm  ON batches  USING GIN (batch_code  gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_students_first_name_trgm ON students USING GIN (first_name  gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_students_last_name_trgm  ON students USING GIN (last_name   gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_courses_course_name_trgm ON courses  USING GIN (course_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_courses_course_code_trgm ON courses  USING GIN (course_code gin_trgm_ops);
