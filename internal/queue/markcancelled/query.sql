-- name: CancelQueueEntryByConsultation :execrows
UPDATE queue_entries
SET status = 'Cancelled', updated_at = now()
WHERE consultation_id = $1
  AND status IN ('Waiting', 'Called', 'Skipped');

-- Cross-module read: patients_profiles for the patient user channel (R2).
-- name: GetSessionForCancelledEntry :one
SELECT qe.clinic_session_id,
       p.user_id AS patient_user_id
FROM queue_entries qe
JOIN patients_profiles p ON p.id = qe.patient_profile_id
WHERE qe.consultation_id = $1
LIMIT 1;
