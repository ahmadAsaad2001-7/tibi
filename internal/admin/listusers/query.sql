-- name: ListUsers :many
SELECT id, email, role, email_verified_at, created_at, deleted_at
FROM identity_users
WHERE (sqlc.narg('role')::text IS NULL OR role::text = sqlc.narg('role'))
  AND (sqlc.narg('search')::text IS NULL
       OR email ILIKE '%' || sqlc.narg('search') || '%')
  AND deleted_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*)::bigint
FROM identity_users
WHERE (sqlc.narg('role')::text IS NULL OR role::text = sqlc.narg('role'))
  AND (sqlc.narg('search')::text IS NULL
       OR email ILIKE '%' || sqlc.narg('search') || '%')
  AND deleted_at IS NULL;