-- name: ProfileID :one
SELECT id FROM doctors_profiles WHERE user_id = @user_id AND deleted_at IS NULL;

-- name: IsPublished :one
SELECT is_published FROM content_posts
WHERE id = @id AND doctor_profile_id = @doctor_profile_id AND deleted_at IS NULL;

-- name: Unpublish :execrows
UPDATE content_posts
SET is_published = false, published_at = NULL, updated_at = @updated_at
WHERE id = @id AND doctor_profile_id = @doctor_profile_id AND deleted_at IS NULL AND is_published = true;