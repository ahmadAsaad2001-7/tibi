-- name: GetProtocolForCancel :one
SELECT id, doctor_profile_id, cancelled_at, xmin::text AS xmin
FROM care_plan_protocols
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetDoctorProfileIDForUser :one
SELECT id FROM doctors_profiles
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: CancelProtocol :exec
UPDATE care_plan_protocols
SET cancelled_at = $2, updated_at = $3
WHERE id = $1;
