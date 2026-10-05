-- name: GetDoctorProfileIDForSchedule :one
SELECT id FROM doctors_profiles
WHERE user_id = @user_id AND deleted_at IS NULL;

-- name: DeleteAllBlocks :exec
DELETE FROM doctors_weekly_schedules WHERE doctor_profile_id = @doctor_profile_id;

-- name: InsertBlock :exec
INSERT INTO doctors_weekly_schedules (
    doctor_profile_id, day_of_week, start_time, end_time,
    slot_duration_minutes, is_active
) VALUES (
             @doctor_profile_id, @day_of_week, @start_time::time, @end_time::time,
             @slot_duration_minutes, @is_active
         );