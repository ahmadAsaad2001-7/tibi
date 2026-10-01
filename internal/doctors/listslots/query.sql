-- name: GetVerifiedDoctor :one
SELECT id FROM doctors_profiles
WHERE id = $1 AND verification_status = 'Verified' AND deleted_at IS NULL;

-- name: BlocksForDoctor :many
SELECT day_of_week, start_time, end_time, slot_duration_minutes, is_active
FROM doctors_weekly_schedules
WHERE doctor_profile_id = $1 AND is_active;

-- name: ExceptionsOnDate :many
SELECT exception_date, from_time, to_time, type
FROM doctors_schedule_exceptions
WHERE doctor_profile_id = $1
  AND exception_date = $2
  AND type = 'Closed';
