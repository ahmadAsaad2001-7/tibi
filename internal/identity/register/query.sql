-- name: FindUserByEmail :one
SELECT id, email, password_hash, role, profile_image_url, created_at
FROM identity_users
WHERE email = $1 AND deleted_at IS NULL;

-- name: InsertUser :one
INSERT INTO identity_users (email, password_hash, role)
VALUES ($1, $2, $3)
RETURNING id, created_at;

-- name: InsertRefreshToken :exec
INSERT INTO identity_refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3);