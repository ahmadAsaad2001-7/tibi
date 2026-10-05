-- name: ListNotifications :many
SELECT id, user_id, type, title, body, payload, is_read, read_at, created_at
FROM communication_notifications
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3;

-- name: CountNotifications :one
SELECT COUNT(*)::bigint
FROM communication_notifications
WHERE user_id = $1 AND deleted_at IS NULL;
