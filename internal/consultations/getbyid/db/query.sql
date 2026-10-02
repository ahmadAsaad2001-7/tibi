-- name: GetByID :one
SELECT c.id, c.status, c.scheduled_at, c.duration_minutes, c.is_urgent, c.notes,
       c.doctor_profile_id, d.full_name AS doctor_name,
       c.patient_profile_id, p.full_name AS patient_name,
       p.user_id AS patient_user_id, d.user_id AS doctor_user_id, c.created_at
FROM consultations_consultations c
JOIN doctors_profiles d ON d.id = c.doctor_profile_id
JOIN patients_profiles p ON p.id = c.patient_profile_id
WHERE c.id = $1 AND c.deleted_at IS NULL;
