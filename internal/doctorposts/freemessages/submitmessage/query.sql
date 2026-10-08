-- name: GetDoctorForFreeMessage :one
SELECT id FROM doctors_profiles
WHERE id = $1 AND verification_status = 'Verified' AND deleted_at IS NULL;

-- name: InsertFreeMessage :one
INSERT INTO doctorposts_free_messages (
    doctor_profile_id, sender_name, sender_email, sender_phone,
    content, sender_ip
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, created_at;