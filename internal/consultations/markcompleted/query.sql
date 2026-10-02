-- name: GetConsultationForComplete :one
SELECT id, status, updated_at, xmin::text AS xmin
FROM consultations_consultations
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateConsultationCompleted :execrows
UPDATE consultations_consultations
SET status = 'Completed', updated_at = $2
WHERE id = $1 AND xmin::text = sqlc.arg('xmin')::text;