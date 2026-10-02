-- name: SoftDeleteFile :one
UPDATE platform_files
SET deleted_at = now()
WHERE id = $1 AND uploader_id = $2 AND deleted_at IS NULL
    RETURNING object_key;