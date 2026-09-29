-- name: FindUserByEmail :one
SELECT id, email, password_hash, role, profile_image_url, created_at
FROM identity_users
WHERE email = $1 AND deleted_at IS NULL;