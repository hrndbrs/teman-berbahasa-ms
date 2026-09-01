-- name: GetStudentByID :one
SELECT id, first_name, last_name, email, phone, date_of_birth, gender,
       address, parent_name, parent_phone, registration_date, status, created_at, updated_at
FROM students
WHERE id = $1;

-- name: GetStudentByEmail :one
SELECT id, first_name, last_name, email, phone, date_of_birth, gender,
       address, parent_name, parent_phone, registration_date, status, created_at, updated_at
FROM students
WHERE email = $1;

-- name: ListStudents :many
SELECT id, first_name, last_name, email, phone, date_of_birth, gender,
       address, parent_name, parent_phone, registration_date, status, created_at, updated_at
FROM students
WHERE
  (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')) AND
  (sqlc.narg('search')::text IS NULL OR (
    first_name ILIKE '%' || sqlc.narg('search') || '%' OR
    last_name  ILIKE '%' || sqlc.narg('search') || '%' OR
    email      ILIKE '%' || sqlc.narg('search') || '%'
  ))
ORDER BY created_at DESC
LIMIT  sqlc.arg('page_size')
OFFSET sqlc.arg('page_offset');

-- name: CountStudents :one
SELECT COUNT(*)
FROM students
WHERE
  (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')) AND
  (sqlc.narg('search')::text IS NULL OR (
    first_name ILIKE '%' || sqlc.narg('search') || '%' OR
    last_name  ILIKE '%' || sqlc.narg('search') || '%' OR
    email      ILIKE '%' || sqlc.narg('search') || '%'
  ));

-- name: GetStudentBatchCodes :many
SELECT e.student_id, b.batch_code
FROM enrollments e
JOIN batches b ON e.batch_id = b.id
WHERE e.student_id = ANY(sqlc.arg('student_ids')::uuid[])
  AND e.status != 'dropped'
ORDER BY e.student_id, b.batch_code;

-- name: GetStudentEnrollments :many
SELECT
  e.id, e.status, e.payment_status, e.enrollment_date,
  b.id    AS batch_id,   b.batch_name, b.batch_code, b.status AS batch_status,
  c.id    AS course_id,  c.course_name, c.course_code, c.level
FROM enrollments e
JOIN batches b ON e.batch_id = b.id
JOIN courses c  ON e.course_id = c.id
WHERE e.student_id = $1
ORDER BY e.created_at DESC
LIMIT 50;

-- name: CreateStudent :one
INSERT INTO students (
  id, first_name, last_name, email, phone, date_of_birth, gender,
  address, parent_name, parent_phone, registration_date
) VALUES (
  sqlc.arg('id'),
  sqlc.arg('first_name'),
  sqlc.arg('last_name'),
  sqlc.arg('email'),
  sqlc.narg('phone'),
  sqlc.narg('date_of_birth'),
  sqlc.narg('gender'),
  sqlc.narg('address'),
  sqlc.narg('parent_name'),
  sqlc.narg('parent_phone'),
  COALESCE(sqlc.narg('registration_date'), CURRENT_DATE)
)
RETURNING id, first_name, last_name, email, phone, date_of_birth, gender,
          address, parent_name, parent_phone, registration_date, status, created_at, updated_at;

-- name: UpdateStudent :one
UPDATE students SET
  first_name    = COALESCE(sqlc.narg('first_name'),    first_name),
  last_name     = COALESCE(sqlc.narg('last_name'),     last_name),
  email         = COALESCE(sqlc.narg('email'),         email),
  phone         = COALESCE(sqlc.narg('phone'),         phone),
  date_of_birth = COALESCE(sqlc.narg('date_of_birth'), date_of_birth),
  gender        = COALESCE(sqlc.narg('gender'),        gender),
  address       = COALESCE(sqlc.narg('address'),       address),
  parent_name   = COALESCE(sqlc.narg('parent_name'),   parent_name),
  parent_phone  = COALESCE(sqlc.narg('parent_phone'),  parent_phone),
  status        = COALESCE(sqlc.narg('status'),        status),
  updated_at    = NOW()
WHERE id = sqlc.arg('id')
RETURNING id, first_name, last_name, email, phone, date_of_birth, gender,
          address, parent_name, parent_phone, registration_date, status, created_at, updated_at;

-- name: UpdateStudentFull :one
UPDATE students SET
  first_name    = @first_name,
  last_name     = @last_name,
  email         = @email,
  phone         = @phone,
  date_of_birth = @date_of_birth,
  gender        = @gender,
  address       = @address,
  parent_name   = @parent_name,
  parent_phone  = @parent_phone,
  status        = @status,
  updated_at    = NOW()
WHERE id = @id
RETURNING id, first_name, last_name, email, phone, date_of_birth, gender,
          address, parent_name, parent_phone, registration_date, status, created_at, updated_at;
