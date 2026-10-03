-- Cross-module read: joins consultations_consultations (Consultations)
-- with doctors_profiles (Doctors) to verify the caller is the treating
-- doctor. Permitted by R2.

-- name: GetClinicalContext :one
SELECT
    c.id, c.patient_profile_id, c.doctor_profile_id, c.status
FROM consultations_consultations c
         JOIN doctors_profiles d ON d.id = c.doctor_profile_id
WHERE c.id = $1
  AND d.user_id = $2
  AND c.deleted_at IS NULL;