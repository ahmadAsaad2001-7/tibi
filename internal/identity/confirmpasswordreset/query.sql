-- name: GetPasswordResetForUpdate :one
SELECT id, user_id, expires_at, consumed_at,
       xmin::text AS xmin
FROM identity_password_resets
WHERE token_hash = $1
FOR UPDATE;

-- name: ConsumePasswordReset :execrows
UPDATE identity_password_resets
SET consumed_at = now()
WHERE id = $1 AND consumed_at IS NULL AND xmin::text = $2;

-- name: UpdateUserPassword :exec
UPDATE identity_users
SET password_hash = $2, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: RevokeAllRefreshTokens :exec
UPDATE identity_refresh_tokens
SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;