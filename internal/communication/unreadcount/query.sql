-- name: UnreadCount :one
SELECT COUNT(*)::bigint
FROM communication_notifications
WHERE user_id = $1 AND is_read = false AND deleted_at IS NULL;
