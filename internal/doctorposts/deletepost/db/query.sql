-- name: ProfileID :one
SELECT id FROM doctors_profiles WHERE user_id = $1 AND deleted_at IS NULL;

-- name: SoftDelete :execrows
UPDATE content_posts
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND doctor_profile_id = $2 AND deleted_at IS NULL;
