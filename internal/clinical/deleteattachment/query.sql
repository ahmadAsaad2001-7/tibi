-- name: SoftDeleteAttachment :execrows
UPDATE clinical_medical_attachments a
SET deleted_at = now(), updated_at = now()
FROM clinical_medical_records r, consultations_consultations c, doctors_profiles d
WHERE a.id = $1
  AND a.medical_record_id = r.id
  AND r.consultation_id = c.id
  AND c.doctor_profile_id = d.id
  AND d.user_id = $2
  AND a.deleted_at IS NULL;
