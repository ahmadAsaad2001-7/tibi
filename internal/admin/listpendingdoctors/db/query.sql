-- name: ListPendingDoctors :many
SELECT
    d.id, d.user_id, d.full_name, d.bio,
    d.consultation_fee::text AS consultation_fee,
    d.currency, d.clinic_name, d.medical_license_number,
    d.submitted_at,
    u.email
FROM doctors_profiles d
         JOIN identity_users u ON u.id = d.user_id
WHERE d.verification_status = 'PendingReview'
  AND d.deleted_at IS NULL
ORDER BY d.submitted_at ASC;