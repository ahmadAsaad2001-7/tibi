-- name: ListMessages :many
SELECT
    m.id, m.consultation_id, m.sender_user_id, m.content,
    m.sent_at, m.read_at
FROM communication_messages m
JOIN consultations_consultations c ON c.id = m.consultation_id
JOIN patients_profiles p ON p.id = c.patient_profile_id
JOIN doctors_profiles  d ON d.id = c.doctor_profile_id
WHERE m.consultation_id = $1
  AND m.deleted_at IS NULL
  AND (p.user_id = $2 OR d.user_id = $2)
ORDER BY m.sent_at DESC, m.id DESC
LIMIT $3 OFFSET $4;

-- name: CountMessages :one
SELECT COUNT(*)::bigint
FROM communication_messages m
JOIN consultations_consultations c ON c.id = m.consultation_id
JOIN patients_profiles p ON p.id = c.patient_profile_id
JOIN doctors_profiles  d ON d.id = c.doctor_profile_id
WHERE m.consultation_id = $1
  AND m.deleted_at IS NULL
  AND (p.user_id = $2 OR d.user_id = $2);

-- name: MarkMessagesRead :exec
UPDATE communication_messages
SET read_at = now()
WHERE consultation_id = $1
  AND sender_user_id != $2
  AND read_at IS NULL
  AND deleted_at IS NULL;
