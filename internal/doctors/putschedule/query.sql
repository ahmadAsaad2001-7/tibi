-- name: GetDoctorProfileIDForSchedule :one
SELECT id FROM doctors_profiles
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: DeleteAllBlocks :exec
DELETE FROM doctors_weekly_schedules WHERE doctor_profile_id = $1;

-- name: InsertBlock :exec
INSERT INTO doctors_weekly_schedules (
    doctor_profile_id, day_of_week, start_time, end_time,
    slot_duration_minutes, is_active
) VALUES ($1, $2, $3::time, $4::time, $5, $6);