-- name: GetDoctorProfileID :one
SELECT id FROM doctors_profiles
WHERE user_id = @user_id AND deleted_at IS NULL;

-- name: InsertException :one
INSERT INTO doctors_schedule_exceptions (
    doctor_profile_id, exception_date, from_time, to_time, type, reason
) VALUES (
             @doctor_profile_id,
             @exception_date,
             @from_time::time,
             @to_time::time,
             @type,
             @reason
         )
    RETURNING id;