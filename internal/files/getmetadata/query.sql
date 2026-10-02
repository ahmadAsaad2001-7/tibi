-- name: GetFileMetadataByID :one
SELECT id, uploader_id, scope, object_key, original_name,
       content_type, size_bytes, content_sha256, created_at, deleted_at
FROM platform_files
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetFileForAuth :one
SELECT id, uploader_id, scope
FROM platform_files
WHERE id = $1 AND deleted_at IS NULL;