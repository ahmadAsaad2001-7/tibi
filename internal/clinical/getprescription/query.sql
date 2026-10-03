-- name: GetPrescriptionForRead :one
SELECT r.id, r.consultation_id, r.issued_at, r.created_at, r.updated_at
FROM clinical_prescriptions r
         JOIN consultations_consultations c ON c.id = r.consultation_id
         LEFT JOIN patients_profiles p ON p.id = c.patient_profile_id AND p.user_id = $2
         LEFT JOIN doctors_profiles  d ON d.id = c.doctor_profile_id  AND d.user_id = $2
WHERE r.consultation_id = $1
  AND r.deleted_at IS NULL
  AND (p.id IS NOT NULL OR d.id IS NOT NULL);

-- name: ListMedications :many
SELECT id, medication_name, dosage, frequency, duration_days, notes
FROM clinical_prescribed_medications
WHERE prescription_id = $1 AND deleted_at IS NULL
ORDER BY id ASC;