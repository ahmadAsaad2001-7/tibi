-- name: GetPostAndUpdateView :one
WITH updated_post AS (
    UPDATE content_posts
    SET view_count = view_count + 1
    WHERE id = sqlc.arg('id')::bigint
      AND deleted_at IS NULL
    RETURNING id, title, excerpt, type, cover_image_url, view_count, like_count, published_at, doctor_profile_id
)
SELECT
    up.id, up.title, up.excerpt, up.type, up.cover_image_url,
    up.view_count, up.like_count, up.published_at,
    d.id AS doctor_id, d.full_name AS doctor_full_name,
    u.profile_image_url AS doctor_profile_image_url
FROM updated_post up
JOIN doctors_profiles d ON d.id = up.doctor_profile_id AND d.deleted_at IS NULL
JOIN identity_users  u ON u.id = d.user_id AND u.deleted_at IS NULL;

-- name: ListFeedByDoctor :many
SELECT
    p.id, p.title, p.excerpt, p.type, p.cover_image_url,
    p.view_count, p.like_count, p.published_at,
    d.id AS doctor_id, d.full_name AS doctor_full_name,
    u.profile_image_url AS doctor_profile_image_url
FROM content_posts p
JOIN doctors_profiles d ON d.id = p.doctor_profile_id AND d.deleted_at IS NULL
JOIN identity_users  u ON u.id = d.user_id AND u.deleted_at IS NULL
WHERE p.is_published = true 
  AND p.deleted_at IS NULL
  AND p.doctor_profile_id = sqlc.arg('doctor_profile_id')::bigint
  AND (sqlc.narg('cursor_published_at')::timestamptz IS NULL
       OR (p.published_at, p.id) < (sqlc.narg('cursor_published_at')::timestamptz, sqlc.narg('cursor_id')::bigint))
ORDER BY p.published_at DESC, p.id DESC
LIMIT sqlc.arg('limit_count');

-- (SpecialtiesForPostAuthors remains the same as before and will be reused)