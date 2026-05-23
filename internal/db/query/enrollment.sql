-- name: GetBatchForEnrollment :one
SELECT b.id, b.status, b.course_id, c.max_capacity
FROM batches b
JOIN courses c ON c.id = b.course_id
WHERE b.id = sqlc.arg('batch_id');

-- name: GetEnrollmentByID :one
SELECT
    e.id, e.status, e.payment_status, e.final_grade,
    e.enrollment_date, e.created_at, e.updated_at,
    e.student_id,
    s.first_name AS student_first_name,
    s.last_name  AS student_last_name,
    s.email      AS student_email,
    e.batch_id,
    b.batch_name, b.batch_code,
    b.status     AS batch_status,
    e.course_id,
    c.course_name, c.course_code
FROM enrollments e
JOIN students s ON s.id = e.student_id
JOIN batches  b ON b.id = e.batch_id
JOIN courses  c ON c.id = e.course_id
WHERE e.id = sqlc.arg('id');

-- name: ListEnrollments :many
SELECT
    e.id, e.status, e.payment_status, e.final_grade,
    e.enrollment_date, e.created_at, e.updated_at,
    e.student_id,
    s.first_name AS student_first_name,
    s.last_name  AS student_last_name,
    s.email      AS student_email,
    e.batch_id,
    b.batch_name, b.batch_code,
    b.status     AS batch_status,
    e.course_id,
    c.course_name, c.course_code
FROM enrollments e
JOIN students s ON s.id = e.student_id
JOIN batches  b ON b.id = e.batch_id
JOIN courses  c ON c.id = e.course_id
WHERE
    (sqlc.narg('batch_id')::uuid      IS NULL OR e.batch_id       = sqlc.narg('batch_id')) AND
    (sqlc.narg('student_id')::uuid    IS NULL OR e.student_id     = sqlc.narg('student_id')) AND
    (sqlc.narg('status')::text        IS NULL OR e.status         = sqlc.narg('status')) AND
    (sqlc.narg('payment_status')::text IS NULL OR e.payment_status = sqlc.narg('payment_status'))
ORDER BY e.created_at DESC
LIMIT  sqlc.arg('page_size')
OFFSET sqlc.arg('page_offset');

-- name: CountEnrollments :one
SELECT COUNT(*)::bigint
FROM enrollments e
WHERE
    (sqlc.narg('batch_id')::uuid       IS NULL OR e.batch_id       = sqlc.narg('batch_id')) AND
    (sqlc.narg('student_id')::uuid     IS NULL OR e.student_id     = sqlc.narg('student_id')) AND
    (sqlc.narg('status')::text         IS NULL OR e.status         = sqlc.narg('status')) AND
    (sqlc.narg('payment_status')::text IS NULL OR e.payment_status = sqlc.narg('payment_status'));

-- name: CreateEnrollment :one
INSERT INTO enrollments (id, student_id, batch_id, course_id)
VALUES (
    sqlc.arg('id'),
    sqlc.arg('student_id'),
    sqlc.arg('batch_id'),
    sqlc.arg('course_id')
)
RETURNING *;

-- name: UpdateEnrollment :one
UPDATE enrollments SET
    status         = COALESCE(sqlc.narg('status'),         status),
    payment_status = COALESCE(sqlc.narg('payment_status'), payment_status),
    final_grade    = COALESCE(sqlc.narg('final_grade'),    final_grade),
    updated_at     = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;
