-- name: GetDoctorProfileIDForUser :one
SELECT id FROM doctors_profiles
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: ListInbox :many
SELECT id, sender_name, sender_email, sender_phone, content,
       is_replied_to, replied_at, reply_content, created_at
FROM doctorposts_free_messages
WHERE doctor_profile_id = $1
  AND (sqlc.narg('unreplied_only')::boolean IS NULL
       OR sqlc.narg('unreplied_only')::boolean = false
       OR is_replied_to = false)
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3;

-- name: CountInbox :one
SELECT COUNT(*)::bigint
FROM doctorposts_free_messages
WHERE doctor_profile_id = $1
  AND (sqlc.narg('unreplied_only')::boolean IS NULL
       OR sqlc.narg('unreplied_only')::boolean = false
       OR is_replied_to = false);