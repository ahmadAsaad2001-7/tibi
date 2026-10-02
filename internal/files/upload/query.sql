-- name: InsertFile :one
INSERT INTO platform_files (
    uploader_id, scope, object_key, original_name,
    content_type, size_bytes, content_sha256
) VALUES ($1, $2, $3, $4, $5, $6, $7)
    RETURNING id, created_at;