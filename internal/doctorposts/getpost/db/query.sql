-- name: GetAndCountView :one
WITH updated AS (
    UPDATE content_posts
    SET view_count = view_count + 1
    WHERE id = sqlc.arg('id')::bigint
      AND is_published
      AND deleted_at IS NULL
    RETURNING id, title, content, COALESCE(excerpt, '') AS excerpt, type, cover_image_url,
              view_count, like_count, COALESCE(published_at, created_at) AS published_at, doctor_profile_id
)
SELECT
    up.id, up.title, up.content, up.excerpt, up.type, up.cover_image_url,
    up.view_count, up.like_count, up.published_at,
    d.id AS doctor_id, d.full_name AS doctor_full_name,
    u.profile_image_url AS doctor_profile_image_url
FROM updated up
JOIN doctors_profiles d ON d.id = up.doctor_profile_id AND d.deleted_at IS NULL
JOIN identity_users  u ON u.id = d.user_id AND u.deleted_at IS NULL;

-- name: Specialties :many
SELECT s.name
FROM doctors_profile_specialties dps
JOIN doctors_specialties s ON s.id = dps.specialty_id
WHERE dps.doctor_profile_id = $1
ORDER BY s.name;
