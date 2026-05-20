-- name: GetUserByIDFull :one
SELECT id, first_name, last_name, email, role, phone, status, created_at, updated_at
FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT id, first_name, last_name, email, role, phone, status, created_at, updated_at
FROM users
WHERE
  (sqlc.narg('role')::text   IS NULL OR role   = sqlc.narg('role'))   AND
  (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC
LIMIT  sqlc.arg('page_size')
OFFSET sqlc.arg('page_offset');

-- name: CountUsers :one
SELECT COUNT(*)
FROM users
WHERE
  (sqlc.narg('role')::text   IS NULL OR role   = sqlc.narg('role'))   AND
  (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'));

-- name: CreateUser :one
INSERT INTO users (id, first_name, last_name, email, password_hash, role, phone)
VALUES (
  sqlc.arg('id'),
  sqlc.arg('first_name'),
  sqlc.arg('last_name'),
  sqlc.arg('email'),
  sqlc.arg('password_hash'),
  sqlc.arg('role'),
  sqlc.narg('phone')
)
RETURNING id, first_name, last_name, email, role, phone, status, created_at, updated_at;

-- name: UpdateUser :one
UPDATE users SET
  first_name = COALESCE(sqlc.narg('first_name'), first_name),
  last_name  = COALESCE(sqlc.narg('last_name'),  last_name),
  email      = COALESCE(sqlc.narg('email'),      email),
  role       = COALESCE(sqlc.narg('role'),       role),
  phone      = COALESCE(sqlc.narg('phone'),      phone),
  status     = COALESCE(sqlc.narg('status'),     status),
  updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING id, first_name, last_name, email, role, phone, status, created_at, updated_at;

-- name: DeleteUserSessions :exec
DELETE FROM refresh_tokens WHERE user_id = $1;
