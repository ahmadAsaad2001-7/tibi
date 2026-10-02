-- name: GetForCancel :one
SELECT c.id, c.status, c.scheduled_at, pp.user_id AS patient_user_id,
       dp.user_id AS doctor_user_id, c.xmin::text AS xmin
FROM consultations_consultations c
JOIN patients_profiles pp ON pp.id = c.patient_profile_id
JOIN doctors_profiles dp ON dp.id = c.doctor_profile_id
WHERE c.id = $1 AND c.deleted_at IS NULL;

-- name: UpdateStatus :execrows
UPDATE consultations_consultations
SET status = $2, updated_at = $3
WHERE id = $1 AND xmin::text = $4 AND deleted_at IS NULL;
