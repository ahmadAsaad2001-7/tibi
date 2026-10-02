-- Cross-module read: joins consultations_consultations (Consultations)
-- with patients_profiles (Patients). Permitted by R2.

-- name: GetConsultationForCheckIn :one
SELECT
    c.id, c.patient_profile_id, c.doctor_profile_id,
    c.clinic_session_id, c.duration_minutes, c.is_urgent, c.status
FROM consultations_consultations c
JOIN patients_profiles p ON p.id = c.patient_profile_id
WHERE c.id = $1
  AND p.user_id = $2
  AND c.status = 'Confirmed'
  AND c.deleted_at IS NULL;