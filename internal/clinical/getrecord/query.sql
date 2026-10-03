-- name: GetRecordForRead :one
SELECT r.id, r.consultation_id, r.patient_profile_id, r.doctor_profile_id,
       r.allergies, r.current_medications, r.past_conditions, r.doctor_notes,
       r.created_at, r.updated_at
FROM clinical_medical_records r
         JOIN consultations_consultations c ON c.id = r.consultation_id
         LEFT JOIN patients_profiles p ON p.id = c.patient_profile_id AND p.user_id = $2
         LEFT JOIN doctors_profiles  d ON d.id = c.doctor_profile_id  AND d.user_id = $2
WHERE r.consultation_id = $1
  AND r.deleted_at IS NULL
  AND (p.id IS NOT NULL OR d.id IS NOT NULL);

-- name: ListAttachments :many
SELECT id, file_id, label, created_at
FROM clinical_medical_attachments
WHERE medical_record_id = $1 AND deleted_at IS NULL
ORDER BY created_at ASC;