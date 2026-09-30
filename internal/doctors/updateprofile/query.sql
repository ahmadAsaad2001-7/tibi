-- name: GetDoctorProfileByUserID :one
SELECT
    id, user_id, full_name, bio,
    consultation_fee::text        AS consultation_fee,
    currency,
    clinic_name, clinic_address, medical_license_number,
    verification_status,
    verification_rejection_reason,
    submitted_at,
    average_rating::text          AS average_rating,
    rating_count,
    created_at, updated_at, deleted_at,
    xmin::text                    AS xmin
FROM doctors_profiles
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: GetSpecialtyIDsForDoctor :many
SELECT specialty_id
FROM doctors_profile_specialties
WHERE doctor_profile_id = $1;

-- name: SpecialtyIDsExist :one
SELECT COUNT(*)::bigint AS count
FROM doctors_specialties
WHERE id = ANY($1::bigint[]);

-- name: UpdateDoctorProfile :execrows
UPDATE doctors_profiles
SET
    bio                    = $2,
    consultation_fee       = $3::numeric,
    currency               = $4,
    clinic_name            = $5,
    clinic_address         = $6,
    medical_license_number = $7,
    updated_at             = now()
WHERE id = $1 AND xmin::text = $8;

-- name: DeleteDoctorSpecialties :exec
DELETE FROM doctors_profile_specialties WHERE doctor_profile_id = $1;

-- name: InsertDoctorSpecialties :copyfrom
INSERT INTO doctors_profile_specialties (doctor_profile_id, specialty_id)
VALUES ($1, $2);