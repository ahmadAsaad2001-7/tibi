-- name: DeleteException :execrows
DELETE FROM doctors_schedule_exceptions e
USING doctors_profiles p
WHERE e.id = $1
  AND e.doctor_profile_id = p.id
  AND p.user_id = $2
  AND p.deleted_at IS NULL;
