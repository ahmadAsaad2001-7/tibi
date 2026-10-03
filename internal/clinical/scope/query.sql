-- name: IsUserParticipantOfFileConsultation :one
SELECT EXISTS (
    SELECT 1
    FROM clinical_medical_attachments a
             JOIN clinical_medical_records r ON r.id = a.medical_record_id
             JOIN consultations_consultations c ON c.id = r.consultation_id
             LEFT JOIN patients_profiles p ON p.id = c.patient_profile_id AND p.user_id = $1
             LEFT JOIN doctors_profiles  d ON d.id = c.doctor_profile_id  AND d.user_id = $1
    WHERE a.file_id = $2
      AND a.deleted_at IS NULL
      AND c.deleted_at IS NULL
      AND (p.id IS NOT NULL OR d.id IS NOT NULL)
) AS is_participant;