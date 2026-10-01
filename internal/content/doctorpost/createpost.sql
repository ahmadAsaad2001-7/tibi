-- name: InsertPost :one
INSERT INTO content_posts (
    doctor_profile_id, title, content, excerpt, type, cover_image_url
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, created_at;