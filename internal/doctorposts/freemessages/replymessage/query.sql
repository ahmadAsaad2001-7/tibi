-- name: GetDoctorProfileIDForUser :one
SELECT id FROM doctors_profiles
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: GetFreeMessageForReply :one
SELECT id, doctor_profile_id, sender_name, sender_email, content,
       is_replied_to, replied_at, reply_content,
       xmin::text AS xmin
FROM doctorposts_free_messages
WHERE id = $1;

-- name: UpdateReply :execrows
UPDATE doctorposts_free_messages
SET is_replied_to = true,
    replied_at = $2,
    reply_content = $3,
    updated_at = now()
WHERE id = $1 AND xmin::text = $4;