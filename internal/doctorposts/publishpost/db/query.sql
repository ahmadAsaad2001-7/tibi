-- name: ProfileID :one
SELECT id FROM doctors_profiles WHERE user_id = $1 AND deleted_at IS NULL;

-- name: IsPublished :one
SELECT is_published FROM content_posts
WHERE id = $1 AND doctor_profile_id = $2 AND deleted_at IS NULL;

-- name: Publish :execrows
UPDATE content_posts
SET is_published = true, published_at = $3, updated_at = $3
WHERE id = $1 AND doctor_profile_id = $2 AND deleted_at IS NULL AND is_published = false;
