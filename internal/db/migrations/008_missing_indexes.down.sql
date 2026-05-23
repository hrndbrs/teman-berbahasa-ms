DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_status;

DROP INDEX IF EXISTS idx_students_created_at;

DROP INDEX IF EXISTS idx_courses_created_at;
DROP INDEX IF EXISTS idx_courses_level;

DROP INDEX IF EXISTS idx_batches_created_at;
DROP INDEX IF EXISTS idx_batches_status_created_at;
DROP INDEX IF EXISTS idx_batches_course_id_created_at;

DROP INDEX IF EXISTS idx_enrollments_course_id_status;

DROP INDEX IF EXISTS idx_batches_batch_name_trgm;
DROP INDEX IF EXISTS idx_batches_batch_code_trgm;
DROP INDEX IF EXISTS idx_students_first_name_trgm;
DROP INDEX IF EXISTS idx_students_last_name_trgm;
DROP INDEX IF EXISTS idx_courses_course_name_trgm;
DROP INDEX IF EXISTS idx_courses_course_code_trgm;
