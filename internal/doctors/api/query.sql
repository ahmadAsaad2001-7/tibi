-- name: SetDoctorVerificationStatus :execrows
UPDATE doctors_profiles
SET verification_status            = $2,
    verification_rejection_reason  = $3,
    submitted_at                   = CASE WHEN $2::verification_status = 'NotSubmitted'
                                          THEN NULL
                                          ELSE submitted_at END,
    updated_at                     = now()
WHERE user_id = $1 AND deleted_at IS NULL;