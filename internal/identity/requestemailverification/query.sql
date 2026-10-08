-- name: GetUserForVerification :one
SELECT id, email, email_verified_at
FROM identity_users
WHERE id = $1 AND deleted_at IS NULL;

-- name: InvalidateOpenVerifications :exec
UPDATE identity_email_verifications
SET consumed_at = now()
WHERE user_id = $1 AND consumed_at IS NULL;

-- name: InsertEmailVerification :exec
INSERT INTO identity_email_verifications (user_id, token_hash, expires_at)
VALUES ($1, $2, $3);