-- name: GetDoctorProfileByUserID :one
SELECT
    id, user_id, full_name, bio,
    consultation_fee::text AS consultation_fee,
    currency,
    clinic_name, clinic_address, medical_license_number,
    verification_status,
    verification_rejection_reason,
    submitted_at,
    average_rating::text AS average_rating,
    rating_count,
    created_at, updated_at, deleted_at,
    xmin::text AS xmin
FROM doctors_profiles
WHERE user_id = @user_id AND deleted_at IS NULL;

-- name: GetSpecialtyIDsForDoctor :many
SELECT specialty_id
FROM doctors_profile_specialties
WHERE doctor_profile_id = @doctor_profile_id;

-- name: SpecialtyIDsExist :one
SELECT COUNT(*)::bigint AS count
FROM doctors_specialties
WHERE id = ANY(@specialty_ids::bigint[]);

-- name: UpdateDoctorProfile :execrows
UPDATE doctors_profiles
SET
    bio = @bio,
    consultation_fee = @consultation_fee::numeric,
    currency = @currency,
    clinic_name = @clinic_name,
    clinic_address = @clinic_address,
    medical_license_number = @medical_license_number,
    updated_at = now()
WHERE id = @id AND xmin::text = @xmin;

-- name: DeleteDoctorSpecialties :exec
DELETE FROM doctors_profile_specialties WHERE doctor_profile_id = @doctor_profile_id;

-- name: InsertDoctorSpecialties :copyfrom
INSERT INTO doctors_profile_specialties (doctor_profile_id, specialty_id)
VALUES ($1, $2);