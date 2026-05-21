-- name: ListBatches :many
SELECT * FROM batches_with_stats
WHERE
    (sqlc.narg('status')::text    IS NULL OR status    = sqlc.narg('status')) AND
    (sqlc.narg('course_id')::uuid IS NULL OR course_id = sqlc.narg('course_id')) AND
    (
        sqlc.narg('search')::text IS NULL OR
        batch_name ILIKE '%' || sqlc.narg('search') || '%' OR
        batch_code ILIKE '%' || sqlc.narg('search') || '%'
    )
ORDER BY created_at DESC
LIMIT  sqlc.arg('page_size')
OFFSET sqlc.arg('page_offset');

-- name: CountBatches :one
SELECT COUNT(*)::bigint
FROM batches
WHERE
    (sqlc.narg('status')::text    IS NULL OR status    = sqlc.narg('status')) AND
    (sqlc.narg('course_id')::uuid IS NULL OR course_id = sqlc.narg('course_id')) AND
    (
        sqlc.narg('search')::text IS NULL OR
        batch_name ILIKE '%' || sqlc.narg('search') || '%' OR
        batch_code ILIKE '%' || sqlc.narg('search') || '%'
    );

-- name: GetBatchByID :one
SELECT * FROM batches_with_stats WHERE id = sqlc.arg('id');

-- name: CreateBatch :one
INSERT INTO batches (
    id, course_id, instructor_user_id, created_by_user_id,
    batch_name, batch_code, start_date, end_date, academic_year
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('course_id'),
    sqlc.arg('instructor_user_id'),
    sqlc.arg('created_by_user_id'),
    sqlc.arg('batch_name'),
    sqlc.arg('batch_code'),
    sqlc.narg('start_date'),
    sqlc.narg('end_date'),
    sqlc.narg('academic_year')
)
RETURNING *;

-- name: UpdateBatch :one
UPDATE batches SET
    instructor_user_id = COALESCE(sqlc.narg('instructor_user_id'), instructor_user_id),
    batch_name         = COALESCE(sqlc.narg('batch_name'),         batch_name),
    batch_code         = COALESCE(sqlc.narg('batch_code'),         batch_code),
    start_date         = COALESCE(sqlc.narg('start_date'),         start_date),
    end_date           = COALESCE(sqlc.narg('end_date'),           end_date),
    academic_year      = COALESCE(sqlc.narg('academic_year'),      academic_year),
    updated_at         = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: UpdateBatchStatus :one
UPDATE batches SET
    status     = sqlc.arg('status'),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING id, status, updated_at;

-- name: DeleteBatch :exec
DELETE FROM batches WHERE id = sqlc.arg('id');

-- name: CountActiveEnrollments :one
SELECT COUNT(*)::bigint
FROM enrollments
WHERE batch_id = sqlc.arg('batch_id') AND status != 'dropped';

-- name: ExistsActiveTeacher :one
SELECT EXISTS(
    SELECT 1 FROM users
    WHERE id = sqlc.arg('id') AND role = 'teacher' AND status = 'active'
);
