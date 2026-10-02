-- name: ListForUser :many
SELECT c.id, c.status, c.scheduled_at, c.duration_minutes, c.is_urgent,
       c.doctor_profile_id, d.full_name AS doctor_name,
       c.patient_profile_id, p.full_name AS patient_name
FROM consultations_consultations c
JOIN doctors_profiles d ON d.id = c.doctor_profile_id
JOIN patients_profiles p ON p.id = c.patient_profile_id
WHERE c.deleted_at IS NULL
  AND (p.user_id = $1 OR d.user_id = $1)
ORDER BY c.scheduled_at DESC
LIMIT 100;
