-- name: GetDoctorProfileID :one
SELECT id FROM doctors_profiles
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: InsertException :one
INSERT INTO doctors_schedule_exceptions (
    doctor_profile_id, exception_date, from_time, to_time, type, reason
) VALUES ($1, $2, $3::time, $4::time, $5, $6)
RETURNING id;
