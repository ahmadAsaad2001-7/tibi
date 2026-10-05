-- name: MarkNotificationRead :execrows
UPDATE communication_notifications
SET is_read = true, read_at = now(), updated_at = now()
WHERE id = $1 AND user_id = $2 AND is_read = false AND deleted_at IS NULL;
