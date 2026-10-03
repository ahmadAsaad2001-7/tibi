-- name: InsertPostAttachment :exec
INSERT INTO content_post_attachments (doctor_post_id, file_id)
VALUES ($1, $2);