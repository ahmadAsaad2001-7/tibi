-- name: GetOrCreateSession :one
INSERT INTO doctors_clinic_sessions (
    doctor_profile_id, session_date, start_time, end_time
) VALUES ($1, $2, $3::time, $4::time)
    ON CONFLICT (doctor_profile_id, session_date, start_time) DO UPDATE
                                                                     SET end_time = EXCLUDED.end_time
                                                                 WHERE false  -- no-op update to RETURNING on conflict
                                                                     RETURNING id, created_at;

-- name: GetSessionForBlock :one
SELECT id FROM doctors_clinic_sessions
WHERE doctor_profile_id = $1
  AND session_date = $2
  AND start_time = $3::time;

-- name: GetDoctorProfileIDForUser :one
SELECT id FROM doctors_profiles
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: GetDoctorProfileByID :one
SELECT id, verification_status
FROM doctors_profiles
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetDoctorBlockForDate :one
SELECT start_time, end_time, slot_duration_minutes
FROM doctors_weekly_schedules
WHERE doctor_profile_id = $1
  AND day_of_week = EXTRACT(DOW FROM $2::date)
  AND is_active
  AND start_time <= $3::time
  AND end_time >= $4::time
LIMIT 1;

-- name: GetClosedException :one
SELECT id FROM doctors_schedule_exceptions
WHERE doctor_profile_id = $1
  AND exception_date = $2
  AND from_time <= $3::time
  AND to_time >= $4::time
  AND type = 'Closed'
LIMIT 1;    