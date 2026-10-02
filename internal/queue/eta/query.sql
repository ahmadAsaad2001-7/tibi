-- name: AverageCompletedSeconds :one
SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at))), 0)::float8 AS avg_seconds
FROM queue_entries
WHERE queue_window_id = $1
  AND status = 'Completed'
  AND started_at IS NOT NULL
  AND completed_at IS NOT NULL;
