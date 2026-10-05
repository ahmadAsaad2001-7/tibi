-- name: InsertNotification :one
INSERT INTO communication_notifications (
    user_id, type, title, body, payload
) VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at;
