-- name: GetProfileForSubmission :one
SELECT id, user_id, full_name, bio,
       consultation_fee::text AS consultation_fee,
    currency, clinic_name, clinic_address, medical_license_number,
       verification_status, submitted_at,
       xmin::text AS xmin
FROM doctors_profiles
WHERE user_id = @user_id AND deleted_at IS NULL;

-- name: UpdateVerificationStatus :execrows
UPDATE doctors_profiles
SET verification_status = @verification_status,
    submitted_at        = @submitted_at,
    updated_at          = now()
WHERE id = @id AND xmin::text = @xmin;

-- name: ListSpecialtyIDs :many
SELECT specialty_id
FROM doctors_profile_specialties
WHERE doctor_profile_id = @doctor_profile_id;