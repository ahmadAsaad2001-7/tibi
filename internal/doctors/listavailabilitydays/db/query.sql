-- name: GetVerifiedDoctor :one
SELECT id FROM doctors_profiles
WHERE id = $1 AND verification_status = 'Verified' AND deleted_at IS NULL;

-- name: BlocksInRange :many
SELECT day_of_week, start_time, end_time, slot_duration_minutes, is_active
FROM doctors_weekly_schedules
WHERE doctor_profile_id = $1 AND is_active;

-- name: ExceptionsInRange :many
SELECT exception_date, from_time, to_time, type
FROM doctors_schedule_exceptions
WHERE doctor_profile_id = $1
  AND exception_date BETWEEN $2 AND $3
  AND type = 'Closed';

-- name: BookedInRange :many
SELECT scheduled_at
FROM consultations_consultations
WHERE doctor_profile_id = $1
  AND scheduled_at >= $2 AND scheduled_at < $3
  AND deleted_at IS NULL
  AND (
    status IN ('Confirmed', 'InProgress')
    OR (status = 'Pending' AND created_at > now() - INTERVAL '15 minutes')
  );
