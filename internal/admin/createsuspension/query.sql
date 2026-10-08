-- name: InsertSuspension :one
INSERT INTO admin_user_suspensions (
    user_id, reason, from_ts, to_ts, vote_id
) VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at;

-- name: RevokeAllRefreshTokens :exec
UPDATE identity_refresh_tokens
SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;