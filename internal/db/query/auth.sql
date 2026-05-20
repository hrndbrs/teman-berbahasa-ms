-- name: GetUserByEmail :one
SELECT id, first_name, last_name, email, password_hash, role, status, failed_attempts
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, first_name, last_name, email, role, status
FROM users
WHERE id = $1;

-- name: IncrementFailedAttempts :exec
UPDATE users
SET failed_attempts = failed_attempts + 1, updated_at = NOW()
WHERE id = $1;

-- name: ResetFailedAttempts :exec
UPDATE users
SET failed_attempts = 0, updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $1, failed_attempts = 0, updated_at = NOW()
WHERE id = $2;

-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4);

-- name: GetRefreshTokenByHash :one
SELECT id, user_id, expires_at
FROM refresh_tokens
WHERE token_hash = $1;

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens WHERE id = $1;

-- name: DeleteRefreshTokensByUserID :exec
DELETE FROM refresh_tokens WHERE user_id = $1;

-- name: InsertPasswordResetToken :exec
INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4);

-- name: GetPasswordResetTokenByHash :one
SELECT id, user_id, expires_at
FROM password_reset_tokens
WHERE token_hash = $1;

-- name: DeletePasswordResetToken :exec
DELETE FROM password_reset_tokens WHERE id = $1;

-- name: DeletePasswordResetTokensByUserID :exec
DELETE FROM password_reset_tokens WHERE user_id = $1;
