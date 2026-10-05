-- name: SetProfileImageFile :exec
UPDATE identity_users
SET profile_image_file_id = $2, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;