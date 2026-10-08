-- name: GetSuspensionForUpdate :one
SELECT id, user_id, reason, from_ts, to_ts, lifted_at,
       xmin::text AS xmin
FROM admin_user_suspensions
WHERE id = $1
FOR UPDATE;

-- name: LiftSuspension :execrows
UPDATE admin_user_suspensions
SET lifted_at = $2, lifted_by = $3, lift_reason = $4, updated_at = $2
WHERE id = $1 AND xmin::text = $5 AND lifted_at IS NULL;