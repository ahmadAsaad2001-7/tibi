-- name: GetDoctorProfileIDForUser :one
SELECT id FROM doctors_profiles
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: GetProtocolForUpdate :one
SELECT id, doctor_profile_id, title, description, reminder_time,
       cancelled_at, xmin::text AS xmin
FROM care_plan_protocols
WHERE id = $1 AND deleted_at IS NULL;

-- name: HasCompletions :one
SELECT EXISTS(SELECT 1 FROM care_plan_completions WHERE protocol_id = $1) AS frozen;

-- name: UpdateProtocol :execrows
UPDATE care_plan_protocols
SET title = $2, description = $3, reminder_time = $4, updated_at = $5
WHERE id = $1 AND xmin::text = sqlc.arg('xmin')::text;
