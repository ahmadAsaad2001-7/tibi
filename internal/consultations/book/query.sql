-- name: GetPatientProfileForUser :one
SELECT id FROM patients_profiles WHERE user_id = $1 AND deleted_at IS NULL;

-- name: CountActiveSlot :one
-- Does an occupied consultation already exist for this doctor at this time?
-- Includes Pendings created within 15 min.
SELECT COUNT(*)::bigint AS n
FROM consultations_consultations
WHERE doctor_profile_id = $1
  AND scheduled_at = $2
  AND deleted_at IS NULL
  AND (
    status IN ('Confirmed', 'InProgress')
        OR (status = 'Pending' AND created_at > now() - INTERVAL '15 minutes')
    );

-- name: CancelStalePending :exec
-- Lazy cleanup of expired Pending consultations occupying this slot.
UPDATE consultations_consultations
SET status = 'Cancelled', updated_at = now()
WHERE doctor_profile_id = $1
  AND scheduled_at = $2
  AND status = 'Pending'
  AND created_at <= now() - INTERVAL '15 minutes'
  AND deleted_at IS NULL;

-- name: InsertConsultation :one
INSERT INTO consultations_consultations (
    patient_profile_id, doctor_profile_id, clinic_session_id,
    scheduled_at, duration_minutes, is_urgent, notes
) VALUES ($1, $2, $3, $4, $5, $6, $7)
    RETURNING id, created_at;

-- name: GetConsultationForUpdate :one
SELECT id, patient_profile_id, doctor_profile_id, scheduled_at,
       duration_minutes, status, is_urgent, notes, created_at, updated_at,
       xmin::text AS xmin
FROM consultations_consultations
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateConsultationStatus :execrows
UPDATE consultations_consultations
SET status = $2, updated_at = $3
WHERE id = $1 AND xmin::text = $4;