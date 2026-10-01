-- Cross-module read: joins identity_users (Identity) with
-- doctors_profiles and friends (Doctors). Permitted by R2
-- (docs/phase-1/00-decisions.md). Writes never cross modules.

-- name: SearchDoctorsByRating :many
SELECT
    d.id, d.full_name, d.bio, d.clinic_name,
    d.consultation_fee::text AS consultation_fee,
    d.currency,
    d.average_rating::text   AS average_rating,
    d.rating_count,
    u.profile_image_url
FROM doctors_profiles d
         JOIN identity_users u ON u.id = d.user_id AND u.deleted_at IS NULL
WHERE d.deleted_at IS NULL
  AND d.verification_status = 'Verified'
  AND (sqlc.narg('q')::text IS NULL
       OR d.full_name   ILIKE '%' || sqlc.narg('q') || '%'
       OR d.clinic_name ILIKE '%' || sqlc.narg('q') || '%'
       OR d.bio         ILIKE '%' || sqlc.narg('q') || '%')
  AND (sqlc.narg('specialty_ids')::bigint[] IS NULL
       OR EXISTS (
           SELECT 1 FROM doctors_profile_specialties dps
           WHERE dps.doctor_profile_id = d.id
             AND dps.specialty_id = ANY(sqlc.narg('specialty_ids')::bigint[])
       ))
  AND (sqlc.narg('min_fee')::numeric IS NULL OR d.consultation_fee >= sqlc.narg('min_fee'))
  AND (sqlc.narg('max_fee')::numeric IS NULL OR d.consultation_fee <= sqlc.narg('max_fee'))
  AND (sqlc.narg('min_rating')::numeric IS NULL OR d.average_rating >= sqlc.narg('min_rating'))
  AND (sqlc.narg('cursor_rating')::numeric IS NULL
       OR (d.average_rating, d.id) < (sqlc.narg('cursor_rating')::numeric, sqlc.narg('cursor_id')::bigint))
ORDER BY d.average_rating DESC, d.id DESC
    LIMIT sqlc.arg('limit_count');

-- name: SearchDoctorsByFeeAsc :many
-- same WHERE clause, then:
AND (sqlc.narg('cursor_fee')::numeric IS NULL
       OR d.consultation_fee > sqlc.narg('cursor_fee')
       OR (d.consultation_fee = sqlc.narg('cursor_fee') AND d.id < sqlc.narg('cursor_id')))
ORDER BY d.consultation_fee ASC, d.id DESC
LIMIT sqlc.arg('limit_count');

-- name: SearchDoctorsByFeeDesc :many
-- same WHERE clause, then:
AND (sqlc.narg('cursor_fee')::numeric IS NULL
       OR (d.consultation_fee, d.id) < (sqlc.narg('cursor_fee')::numeric, sqlc.narg('cursor_id')::bigint))
ORDER BY d.consultation_fee DESC, d.id DESC
LIMIT sqlc.arg('limit_count');

-- name: SearchDoctorsByCreatedAt :many
-- same WHERE clause, then:
AND (sqlc.narg('cursor_created_at')::timestamptz IS NULL
       OR (d.created_at, d.id) < (sqlc.narg('cursor_created_at')::timestamptz, sqlc.narg('cursor_id')::bigint))
ORDER BY d.created_at DESC, d.id DESC
LIMIT sqlc.arg('limit_count');

-- name: SpecialtiesForDoctors :many
SELECT dps.doctor_profile_id, s.id AS specialty_id, s.name
FROM doctors_profile_specialties dps
         JOIN doctors_specialties s ON s.id = dps.specialty_id
WHERE dps.doctor_profile_id = ANY(sqlc.arg('doctor_ids')::bigint[])
ORDER BY s.name;

-- name: BlocksForDoctors :many
SELECT doctor_profile_id, day_of_week, start_time, end_time,
       slot_duration_minutes, is_active
FROM doctors_weekly_schedules
WHERE doctor_profile_id = ANY(sqlc.arg('doctor_ids')::bigint[])
  AND is_active;

-- name: ExceptionsForDoctors :many
SELECT doctor_profile_id, exception_date, from_time, to_time, type
FROM doctors_schedule_exceptions
WHERE doctor_profile_id = ANY(sqlc.arg('doctor_ids')::bigint[])
  AND exception_date BETWEEN sqlc.arg('from_date') AND sqlc.arg('to_date')
  AND type = 'Closed';

-- name: BookedSlotsForDoctors :many
SELECT doctor_profile_id, scheduled_at
FROM consultations_consultations
WHERE doctor_profile_id = ANY(sqlc.arg('doctor_ids')::bigint[])
  AND scheduled_at BETWEEN sqlc.arg('from_ts') AND sqlc.arg('to_ts')
  AND deleted_at IS NULL
  AND (
    status IN ('Confirmed', 'InProgress')
        OR (status = 'Pending' AND created_at > now() - INTERVAL '15 minutes')
    );