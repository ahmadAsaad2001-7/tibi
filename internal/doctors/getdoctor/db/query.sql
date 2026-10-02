-- Cross-module read; see search/query.sql.

-- name: GetDoctorByID :one
SELECT
    d.id, d.full_name, d.bio, d.clinic_name, d.clinic_address,
    d.consultation_fee::text AS consultation_fee,
    d.currency,
    d.average_rating::text   AS average_rating,
    d.rating_count,
    d.medical_license_number,
    d.verification_status,
    u.profile_image_url
FROM doctors_profiles d
JOIN identity_users u ON u.id = d.user_id AND u.deleted_at IS NULL
WHERE d.id = $1
  AND d.deleted_at IS NULL
  AND d.verification_status = 'Verified';

-- name: SpecialtiesForDoctor :many
SELECT s.id, s.name
FROM doctors_profile_specialties dps
JOIN doctors_specialties s ON s.id = dps.specialty_id
WHERE dps.doctor_profile_id = $1
ORDER BY s.name;

-- name: BlocksForDoctor :many
SELECT day_of_week, start_time, end_time, slot_duration_minutes, is_active
FROM doctors_weekly_schedules
WHERE doctor_profile_id = $1 AND is_active;

-- name: ExceptionsForDoctor :many
SELECT exception_date, from_time, to_time, type
FROM doctors_schedule_exceptions
WHERE doctor_profile_id = $1
  AND exception_date BETWEEN $2 AND $3
  AND type = 'Closed';

-- name: BookedSlots :many
SELECT scheduled_at
FROM consultations_consultations
WHERE doctor_profile_id = $1
  AND scheduled_at >= $2 AND scheduled_at < $3
  AND deleted_at IS NULL
  AND (
    status IN ('Confirmed', 'InProgress')
        OR (status = 'Pending' AND created_at > now() - INTERVAL '15 minutes')
    );

-- name: RecentPosts :many
SELECT id, title, COALESCE(excerpt, '') AS excerpt, type, COALESCE(published_at, created_at) AS published_at
FROM content_posts
WHERE doctor_profile_id = $1
  AND is_published
  AND deleted_at IS NULL
ORDER BY published_at DESC NULLS LAST, id DESC
LIMIT 5;
