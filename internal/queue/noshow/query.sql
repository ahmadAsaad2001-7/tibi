-- name: GetEntryForUpdate :one
SELECT queue_entries.id, consultation_id, queue_window_id, clinic_session_id,
       patient_profile_id, queue_number, is_priority, status,
       checked_in_at, called_at, started_at, completed_at, skip_reason,
       created_at, updated_at, queue_entries.xmin::text AS xmin,
       (SELECT p.user_id FROM patients_profiles p WHERE p.id = queue_entries.patient_profile_id) AS patient_user_id
FROM queue_entries
WHERE queue_entries.id = $1
  AND EXISTS (
      SELECT 1 FROM doctors_clinic_sessions s
      JOIN doctors_profiles d ON d.id = s.doctor_profile_id
      WHERE s.id = queue_entries.clinic_session_id
        AND (
            d.user_id = $2
            OR EXISTS (
                SELECT 1 FROM identity_users u
                WHERE u.id = $2 AND u.role = 'Admin' AND u.deleted_at IS NULL
            )
        )
  )
FOR UPDATE;

-- name: HasOtherActive :one
SELECT EXISTS (
    SELECT 1 FROM queue_entries
    WHERE queue_window_id = $1
      AND status IN ('Called', 'InProgress')
      AND id != $2
) AS has_other;

-- name: UpdateEntryStatus :execrows
UPDATE queue_entries
SET status = $2, called_at = $3, started_at = $4,
    completed_at = $5, skip_reason = $6, updated_at = $7
WHERE id = $1 AND xmin::text = sqlc.arg('xmin')::text;

-- name: SetWindowCurrent :exec
UPDATE queue_windows
SET current_queue_entry_id = $2, updated_at = $3
WHERE id = $1;

-- name: ClearWindowCurrent :exec
UPDATE queue_windows
SET current_queue_entry_id = NULL, updated_at = $2
WHERE id = $1 AND current_queue_entry_id = $3;