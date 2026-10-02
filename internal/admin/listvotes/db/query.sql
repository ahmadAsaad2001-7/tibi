-- name: ListOpenVotes :many
SELECT
    v.id, v.action_type, v.target_user_id, v.status,
    v.required_votes, v.votes_for, v.votes_against,
    v.expires_at, v.created_at,
    u.email        AS target_email,
    u.role         AS target_role,
    COALESCE(d.full_name, p.full_name, '') AS target_full_name
FROM admin_votes v
         JOIN identity_users u ON u.id = v.target_user_id
         LEFT JOIN doctors_profiles  d ON d.user_id = v.target_user_id AND d.deleted_at IS NULL
         LEFT JOIN patients_profiles p ON p.user_id = v.target_user_id AND p.deleted_at IS NULL
WHERE v.status = 'Open' AND v.deleted_at IS NULL
ORDER BY v.expires_at ASC;