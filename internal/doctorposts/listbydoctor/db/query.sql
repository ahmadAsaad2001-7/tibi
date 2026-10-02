-- name: DoctorExists :one
SELECT id FROM doctors_profiles
WHERE id = $1 AND deleted_at IS NULL AND verification_status = 'Verified';

-- name: ListByDoctor :many
SELECT
    p.id, p.title, COALESCE(p.excerpt, '') AS excerpt, p.type, p.cover_image_url,
    p.view_count, p.like_count, COALESCE(p.published_at, p.created_at) AS published_at,
    d.id AS doctor_id, d.full_name AS doctor_full_name,
    u.profile_image_url AS doctor_profile_image_url
FROM content_posts p
JOIN doctors_profiles d ON d.id = p.doctor_profile_id AND d.deleted_at IS NULL
JOIN identity_users  u ON u.id = d.user_id AND u.deleted_at IS NULL
WHERE p.is_published = true
  AND p.deleted_at IS NULL
  AND p.doctor_profile_id = sqlc.arg('doctor_profile_id')::bigint
  AND (sqlc.narg('cursor_published_at')::timestamptz IS NULL
       OR (COALESCE(p.published_at, p.created_at), p.id) < (sqlc.narg('cursor_published_at')::timestamptz, sqlc.narg('cursor_id')::bigint))
ORDER BY COALESCE(p.published_at, p.created_at) DESC, p.id DESC
LIMIT sqlc.arg('limit_count');

-- name: Specialties :many
SELECT s.name
FROM doctors_profile_specialties dps
JOIN doctors_specialties s ON s.id = dps.specialty_id
WHERE dps.doctor_profile_id = $1
ORDER BY s.name;
