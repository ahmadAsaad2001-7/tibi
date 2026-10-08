-- name: ListSuspensions :many
SELECT
    s.id, s.user_id, s.reason, s.from_ts, s.to_ts,
    s.lifted_at, s.lifted_by, s.lift_reason, s.created_at,
    u.email AS user_email,
    u.role  AS user_role
FROM admin_user_suspensions s
JOIN identity_users u ON u.id = s.user_id
WHERE (sqlc.narg('active_only')::boolean IS NULL
       OR sqlc.narg('active_only')::boolean = false
       OR (s.lifted_at IS NULL AND (s.to_ts IS NULL OR s.to_ts > now())))
ORDER BY s.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountSuspensions :one
SELECT COUNT(*)::bigint
FROM admin_user_suspensions s
WHERE (sqlc.narg('active_only')::boolean IS NULL
       OR sqlc.narg('active_only')::boolean = false
       OR (s.lifted_at IS NULL AND (s.to_ts IS NULL OR s.to_ts > now())));