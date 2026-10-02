-- name: LockSession :one
-- Serializes bookings on the same clinic session. Held until commit.
SELECT id FROM doctors_clinic_sessions WHERE id = $1 FOR UPDATE;
