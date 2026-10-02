-- name: ProfileID :one
SELECT id FROM doctors_profiles
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: UpdatePost :one
UPDATE content_posts SET
    title = COALESCE(sqlc.narg('title'), title),
    content = COALESCE(sqlc.narg('content'), content),
    excerpt = COALESCE(sqlc.narg('excerpt'), excerpt),
    type = COALESCE(sqlc.narg('type')::post_type, type),
    cover_image_url = COALESCE(sqlc.narg('cover_image_url'), cover_image_url),
    updated_at = now()
WHERE id = sqlc.arg('id')
  AND doctor_profile_id = sqlc.arg('doctor_profile_id')
  AND deleted_at IS NULL
RETURNING id, updated_at;
