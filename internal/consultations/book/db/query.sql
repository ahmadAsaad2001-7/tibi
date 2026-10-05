-- name: GetPatientProfileForUser :one
SELECT id FROM patients_profiles WHERE user_id = $1 AND deleted_at IS NULL;

-- name: LockSession :one
SELECT id FROM doctors_clinic_sessions WHERE id = $1 FOR UPDATE;

-- name: CancelStalePending :exec
UPDATE consultations_consultations
SET status = 'Cancelled', updated_at = now()
WHERE doctor_profile_id = $1
  AND scheduled_at = $2
  AND status = 'Pending'
  AND created_at <= now() - INTERVAL '15 minutes'
  AND deleted_at IS NULL;

-- name: CountActiveSlot :one
SELECT COUNT(*)::bigint AS n
FROM consultations_consultations
WHERE doctor_profile_id = $1
  AND scheduled_at = $2
  AND deleted_at IS NULL
  AND (
    status IN ('Confirmed', 'InProgress')
        OR (status = 'Pending' AND created_at > now() - INTERVAL '15 minutes')
    );

-- name: InsertConsultation :one
INSERT INTO consultations_consultations (
    patient_profile_id, doctor_profile_id, clinic_session_id,
    scheduled_at, duration_minutes, is_urgent, notes
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, created_at;

-- name: GetDoctorUserID :one
SELECT user_id FROM doctors_profiles WHERE id = $1 AND deleted_at IS NULL;

