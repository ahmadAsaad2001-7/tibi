-- name: GetWindowForUpdate :one
-- Ensures the window exists then locks it. Called AFTER EnsureWindow.
SELECT id, clinic_session_id, status, next_queue_number,
       current_queue_entry_id, xmin::text AS xmin
FROM queue_windows
WHERE clinic_session_id = $1
FOR UPDATE;

-- name: EnsureWindow :one
INSERT INTO queue_windows (clinic_session_id, status)
VALUES ($1, 'Open')
ON CONFLICT (clinic_session_id) DO UPDATE
    SET updated_at = queue_windows.updated_at  -- no-op to fire RETURNING
RETURNING id;

-- name: IncrementQueueNumber :one
UPDATE queue_windows
SET next_queue_number = next_queue_number + 1,
    updated_at = now()
WHERE id = $1
RETURNING next_queue_number - 1 AS assigned_number;

-- name: GetExistingEntryForConsultation :one
SELECT id, queue_number, is_priority, status, checked_in_at, queue_window_id
FROM queue_entries
WHERE consultation_id = $1;

-- name: InsertQueueEntry :one
INSERT INTO queue_entries (
    consultation_id, queue_window_id, clinic_session_id, patient_profile_id,
    queue_number, is_priority, status, checked_in_at
) VALUES ($1, $2, $3, $4, $5, $6, 'Waiting', now())
RETURNING id, created_at;

-- name: CountAhead :one
SELECT COUNT(*)::int AS n
FROM queue_entries
WHERE queue_window_id = sqlc.arg('queue_window_id')
  AND id != sqlc.arg('id')
  AND status IN ('Waiting', 'Called', 'Skipped')
  AND (
      (is_priority = true AND sqlc.arg('is_priority')::boolean = false)
      OR (is_priority = sqlc.arg('is_priority')::boolean AND queue_number < sqlc.arg('queue_number'))
  );
