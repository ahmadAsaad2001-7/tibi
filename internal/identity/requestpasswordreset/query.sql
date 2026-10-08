-- name: GetUserByEmailForReset :one
SELECT id, email FROM identity_users
WHERE email = $1 AND deleted_at IS NULL;

-- name: InvalidateOpenPasswordResets :exec
UPDATE identity_password_resets
SET consumed_at = now()
WHERE user_id = $1 AND consumed_at IS NULL;

-- name: InsertPasswordReset :exec
INSERT INTO identity_password_resets (user_id, token_hash, expires_at)
VALUES ($1, $2, $3);