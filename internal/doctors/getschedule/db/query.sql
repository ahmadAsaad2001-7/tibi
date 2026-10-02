-- name: GetDoctorProfileID :one
SELECT id FROM doctors_profiles
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: ListBlocks :many
SELECT day_of_week, start_time, end_time, slot_duration_minutes, is_active
FROM doctors_weekly_schedules
WHERE doctor_profile_id = $1
ORDER BY day_of_week, start_time;

-- name: ListExceptions :many
SELECT id, exception_date, from_time, to_time, type, reason
FROM doctors_schedule_exceptions
WHERE doctor_profile_id = $1
ORDER BY exception_date, from_time;
