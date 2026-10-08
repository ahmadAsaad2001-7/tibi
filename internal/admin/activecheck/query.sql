-- name: GetActiveSuspension :one
SELECT reason, from_ts, to_ts
FROM admin_user_suspensions
WHERE user_id = $1
  AND lifted_at IS NULL
  AND (to_ts IS NULL OR to_ts > now())
ORDER BY from_ts DESC
LIMIT 1;