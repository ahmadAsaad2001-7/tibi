-- name: GetConsultationParticipants :one
SELECT
    c.id,
    c.patient_profile_id,
    c.doctor_profile_id,
    p.user_id AS patient_user_id,
    d.user_id AS doctor_user_id,
    c.status
FROM consultations_consultations c
JOIN patients_profiles p ON p.id = c.patient_profile_id
JOIN doctors_profiles  d ON d.id = c.doctor_profile_id
WHERE c.id = $1 AND c.deleted_at IS NULL;

-- name: InsertMessage :one
INSERT INTO communication_messages (
    consultation_id, sender_user_id, content, sent_at
) VALUES ($1, $2, $3, $4)
RETURNING id, sent_at, created_at;
