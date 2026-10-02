-- Doctor of the session or admin. Ownership in SQL (anti-enumeration).
-- name: GetWindowBySessionForUpdate :one
SELECT qw.id, qw.status, qw.xmin::text AS xmin
FROM queue_windows qw
JOIN doctors_clinic_sessions s ON s.id = qw.clinic_session_id
JOIN doctors_profiles d ON d.id = s.doctor_profile_id
WHERE qw.clinic_session_id = $1
  AND (
      d.user_id = $2
      OR EXISTS (
          SELECT 1 FROM identity_users u
          WHERE u.id = $2 AND u.role = 'Admin' AND u.deleted_at IS NULL
      )
  )
FOR UPDATE;

-- name: CloseWindow :execrows
UPDATE queue_windows
SET status = 'Closed', updated_at = now()
WHERE id = $1 AND xmin::text = sqlc.arg('xmin')::text;

-- name: FailRemainingEntries :execrows
UPDATE queue_entries
SET status = 'NoShow',
    skip_reason = 'session closed',
    updated_at = now()
WHERE queue_window_id = $1
  AND status IN ('Waiting', 'Called', 'Skipped');
