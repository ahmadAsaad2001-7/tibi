-- name: MarkAllRead :execrows
UPDATE communication_notifications
SET is_read = true, read_at = now(), updated_at = now()
WHERE user_id = $1 AND is_read = false AND deleted_at IS NULL;
