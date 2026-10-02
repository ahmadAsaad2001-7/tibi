-- name: IsQueueMember :one
-- Doctor of the session, admin, or a patient with a non-terminal entry.
SELECT EXISTS (
    SELECT 1 FROM doctors_clinic_sessions s
    JOIN doctors_profiles d ON d.id = s.doctor_profile_id
    JOIN identity_users u ON u.id = d.user_id
    WHERE s.id = sqlc.arg('clinic_session_id') AND u.id = sqlc.arg('user_id')
    UNION ALL
    SELECT 1 FROM queue_entries qe
    JOIN patients_profiles p ON p.id = qe.patient_profile_id
    WHERE qe.clinic_session_id = sqlc.arg('clinic_session_id')
      AND p.user_id = sqlc.arg('user_id')
      AND qe.status NOT IN ('Completed', 'NoShow', 'Cancelled')
    UNION ALL
    SELECT 1 FROM identity_users
    WHERE id = sqlc.arg('user_id') AND role = 'Admin'
) AS is_member;
