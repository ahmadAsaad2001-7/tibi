-- name: GetVerificationForUpdate :one
SELECT id, user_id, expires_at, consumed_at,
       xmin::text AS xmin
FROM identity_email_verifications
WHERE token_hash = $1
FOR UPDATE;

-- name: ConsumeVerification :execrows
UPDATE identity_email_verifications
SET consumed_at = now()
WHERE id = $1 AND consumed_at IS NULL AND xmin::text = $2;

-- name: SetEmailVerified :exec
UPDATE identity_users
SET email_verified_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL AND email_verified_at IS NULL;