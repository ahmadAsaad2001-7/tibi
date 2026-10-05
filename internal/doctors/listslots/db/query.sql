-- name: GetVerifiedDoctor :one
SELECT id FROM doctors_profiles
WHERE id = @id AND verification_status = 'Verified' AND deleted_at IS NULL;

-- name: BlocksForDoctor :many
SELECT day_of_week, start_time, end_time, slot_duration_minutes, is_active
FROM doctors_weekly_schedules
WHERE doctor_profile_id = @doctor_profile_id AND is_active;

-- name: ExceptionsOnDate :many
SELECT exception_date, from_time, to_time, type
FROM doctors_schedule_exceptions
WHERE doctor_profile_id = @doctor_profile_id
  AND exception_date = @exception_date
  AND type = 'Closed';

-- name: BookedOnDate :many
SELECT scheduled_at
FROM consultations_consultations
WHERE doctor_profile_id = @doctor_profile_id
  AND scheduled_at >= @from_ts AND scheduled_at < @to_ts
  AND deleted_at IS NULL
  AND (
    status IN ('Confirmed', 'InProgress')
        OR (status = 'Pending' AND created_at > now() - INTERVAL '15 minutes')
    );