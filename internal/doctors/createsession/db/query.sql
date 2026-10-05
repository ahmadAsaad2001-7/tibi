-- name: GetDoctorProfileByID :one
SELECT id, verification_status
FROM doctors_profiles
WHERE id = @id AND deleted_at IS NULL;

-- name: GetOrCreateSession :one
INSERT INTO doctors_clinic_sessions (
    doctor_profile_id, session_date, start_time, end_time
) VALUES (
             @doctor_profile_id, @session_date, @start_time::time, @end_time::time
         )
    ON CONFLICT (doctor_profile_id, session_date, start_time)
DO UPDATE SET end_time = doctors_clinic_sessions.end_time
           RETURNING id, created_at;

-- name: GetDoctorBlockForDate :one
SELECT start_time, end_time, slot_duration_minutes
FROM doctors_weekly_schedules
WHERE doctor_profile_id = @doctor_profile_id
  AND day_of_week = EXTRACT(DOW FROM @date::date)
  AND is_active
  AND start_time <= @start_time::time
  AND end_time >= @end_time::time
LIMIT 1;

-- name: GetClosedException :one
SELECT id FROM doctors_schedule_exceptions
WHERE doctor_profile_id = @doctor_profile_id
  AND exception_date = @exception_date
  AND from_time <= @from_time::time
  AND to_time >= @to_time::time
  AND type = 'Closed'
LIMIT 1;