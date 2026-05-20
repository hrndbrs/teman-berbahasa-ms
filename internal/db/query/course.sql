-- name: ListCourses :many
SELECT
  id, course_name, course_code, description, subject, level,
  session_count, price, max_capacity, status, created_at, updated_at,
  batch_count, ongoing_batch_count, enrolled_count
FROM courses_with_stats
WHERE
  (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')) AND
  (sqlc.narg('level')::text  IS NULL OR level  = sqlc.narg('level'))  AND
  (
    sqlc.narg('search')::text IS NULL OR
    course_name ILIKE '%' || sqlc.narg('search') || '%' OR
    course_code ILIKE '%' || sqlc.narg('search') || '%'
  )
ORDER BY created_at DESC
LIMIT  sqlc.arg('page_size')
OFFSET sqlc.arg('page_offset');

-- name: CountCourses :one
SELECT COUNT(*)
FROM courses
WHERE
  (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')) AND
  (sqlc.narg('level')::text  IS NULL OR level  = sqlc.narg('level'))  AND
  (
    sqlc.narg('search')::text IS NULL OR
    course_name ILIKE '%' || sqlc.narg('search') || '%' OR
    course_code ILIKE '%' || sqlc.narg('search') || '%'
  );

-- name: GetCourseByID :one
SELECT
  c.id, c.course_name, c.course_code, c.description, c.subject, c.level,
  c.session_count, c.price, c.max_capacity, c.status, c.created_at, c.updated_at,
  COUNT(b.id)::bigint                                       AS batch_count,
  COUNT(b.id) FILTER (WHERE b.status = 'ongoing')::bigint   AS ongoing_batch_count,
  COALESCE((
    SELECT COUNT(*)::bigint
    FROM enrollments e
    WHERE e.course_id = c.id AND e.status != 'dropped'
  ), 0)                                                      AS enrolled_count
FROM courses c
LEFT JOIN batches b ON b.course_id = c.id
WHERE c.id = sqlc.arg('id')
GROUP BY
  c.id, c.course_name, c.course_code, c.description, c.subject,
  c.level, c.session_count, c.price, c.max_capacity, c.status,
  c.created_at, c.updated_at;

-- name: CreateCourse :one
INSERT INTO courses (id, course_name, course_code, description, subject, level, session_count, price, max_capacity)
VALUES (
  sqlc.arg('id'),
  sqlc.arg('course_name'),
  sqlc.arg('course_code'),
  sqlc.narg('description'),
  sqlc.narg('subject'),
  sqlc.narg('level'),
  sqlc.narg('session_count'),
  sqlc.narg('price'),
  sqlc.narg('max_capacity')
)
RETURNING id, course_name, course_code, description, subject, level, session_count, price, max_capacity, status, created_at, updated_at;

-- name: UpdateCourse :one
UPDATE courses SET
  course_name    = COALESCE(sqlc.narg('course_name'),    course_name),
  course_code    = COALESCE(sqlc.narg('course_code'),    course_code),
  description    = COALESCE(sqlc.narg('description'),    description),
  subject        = COALESCE(sqlc.narg('subject'),        subject),
  level          = COALESCE(sqlc.narg('level'),          level),
  session_count = COALESCE(sqlc.narg('session_count'), session_count),
  price          = COALESCE(sqlc.narg('price'),          price),
  max_capacity   = COALESCE(sqlc.narg('max_capacity'),   max_capacity),
  updated_at     = NOW()
WHERE id = sqlc.arg('id')
RETURNING id, course_name, course_code, description, subject, level, session_count, price, max_capacity, status, created_at, updated_at;

-- name: ArchiveCourse :one
UPDATE courses SET
  status     = 'archived',
  updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING id, status, updated_at;

-- name: CountOngoingBatchesByCourse :one
SELECT COUNT(*)::bigint
FROM batches
WHERE course_id = sqlc.arg('course_id') AND status = 'ongoing';
