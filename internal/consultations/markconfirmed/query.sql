-- Cross-module read: patients_profiles and doctors_profiles for WS notify (R2).
-- name: GetConsultationForConfirm :one
SELECT c.id, c.status, c.created_at, c.updated_at, c.xmin::text AS xmin,
       p.user_id AS patient_user_id, d.user_id AS doctor_user_id
FROM consultations_consultations c
JOIN patients_profiles p ON p.id = c.patient_profile_id
JOIN doctors_profiles d ON d.id = c.doctor_profile_id
WHERE c.id = $1 AND c.deleted_at IS NULL;

-- name: UpdateConsultationStatus :execrows
UPDATE consultations_consultations
SET status = $2, updated_at = $3
WHERE id = $1 AND xmin::text = sqlc.arg('xmin')::text;