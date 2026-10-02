-- Cross-module read: joins consultations_consultations (Consultations)
-- with patients_profiles (Patients) and doctors_profiles (Doctors).
-- Permitted by R2. Membership rule for WebSocket subscriptions.

-- name: IsConsultationMember :one
SELECT EXISTS (
    SELECT 1 FROM consultations_consultations c
    LEFT JOIN patients_profiles p ON p.id = c.patient_profile_id AND p.user_id = $2
    LEFT JOIN doctors_profiles  d ON d.id = c.doctor_profile_id  AND d.user_id = $2
    WHERE c.id = $1
      AND c.deleted_at IS NULL
      AND (p.id IS NOT NULL OR d.id IS NOT NULL)
) AS is_member;

-- name: OtherConsultationMember :one
SELECT
    CASE
        WHEN p.user_id = $2 THEN d.user_id
        ELSE p.user_id
    END::bigint AS other_user_id
FROM consultations_consultations c
JOIN patients_profiles p ON p.id = c.patient_profile_id
JOIN doctors_profiles  d ON d.id = c.doctor_profile_id
WHERE c.id = $1
  AND (p.user_id = $2 OR d.user_id = $2);
