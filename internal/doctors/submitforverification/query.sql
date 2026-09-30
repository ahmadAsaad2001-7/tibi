-- name: GetProfileForSubmission :one
SELECT id, user_id, full_name, bio,
       consultation_fee::text AS consultation_fee,
       currency, clinic_name, clinic_address, medical_license_number,
       verification_status, submitted_at,
       xmin::text AS xmin
FROM doctors_profiles
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: UpdateVerificationStatus :execrows
UPDATE doctors_profiles
SET verification_status = $2,
    submitted_at        = $3,
    updated_at          = now()
WHERE id = $1 AND xmin::text = $4;

-- name: CountSpecialties :one
SELECT COUNT(*)::bigint FROM doctors_profile_specialties WHERE doctor_profile_id = $1;

-- name: ListSpecialtyIDs :many
SELECT specialty_id
FROM doctors_profile_specialties
WHERE doctor_profile_id = $1;