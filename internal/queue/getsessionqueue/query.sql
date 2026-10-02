-- Cross-module read: joins queue_entries with patients_profiles for display
-- names, and doctors_clinic_sessions / doctors_profiles / identity_users
-- for ownership. Permitted by R2.

-- name: GetOwnedSession :one
SELECT s.id
FROM doctors_clinic_sessions s
JOIN doctors_profiles d ON d.id = s.doctor_profile_id
WHERE s.id = $1
  AND (
      d.user_id = $2
      OR EXISTS (
          SELECT 1 FROM identity_users u
          WHERE u.id = $2 AND u.role = 'Admin' AND u.deleted_at IS NULL
      )
  );

-- name: GetWindowForSession :one
SELECT id, status, current_queue_entry_id, next_queue_number
FROM queue_windows
WHERE clinic_session_id = $1;

-- name: GetSessionQueue :many
SELECT
    qe.id, qe.queue_number, qe.status, qe.is_priority,
    qe.checked_in_at, qe.called_at, qe.started_at, qe.completed_at,
    p.full_name AS patient_full_name
FROM queue_entries qe
JOIN patients_profiles p ON p.id = qe.patient_profile_id
WHERE qe.clinic_session_id = $1
ORDER BY
    CASE WHEN qe.status IN ('Completed', 'NoShow', 'Cancelled') THEN 1 ELSE 0 END,
    qe.is_priority DESC,
    qe.queue_number ASC;
