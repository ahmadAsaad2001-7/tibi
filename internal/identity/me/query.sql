-- name: GetMe :one
SELECT
    u.id,
    u.email,
    u.role,
    u.profile_image_file_id,  -- ✅ KEPT: Required for the delta's PresignGet logic
    u.profile_image_url,      -- ✅ KEPT: Fallback if file_id is nil or presign fails
    u.created_at,
    p.id                       AS patient_id,
    p.full_name                AS patient_full_name,
    p.phone_number             AS patient_phone_number,
    p.date_of_birth            AS patient_date_of_birth,
    p.insurance_provider       AS patient_insurance_provider,
    p.insurance_policy_number  AS patient_insurance_policy_number,
    d.id                       AS doctor_id,
    -- ❌ REMOVED DUPLICATE: u.profile_image_file_id As profile_image_file_id,
    d.full_name                AS doctor_full_name,
    d.bio                      AS doctor_bio,
    d.consultation_fee::text   AS doctor_consultation_fee,
    d.currency                 AS doctor_currency,
    d.clinic_name              AS doctor_clinic_name,
    d.clinic_address           AS doctor_clinic_address,
    d.medical_license_number   AS doctor_medical_license_number,
    d.verification_status      AS doctor_verification_status,
    d.average_rating::text     AS doctor_average_rating,
    d.rating_count             AS doctor_rating_count
FROM identity_users u
         LEFT JOIN patients_profiles p ON p.user_id = u.id AND p.deleted_at IS NULL
         LEFT JOIN doctors_profiles  d ON d.user_id = u.id AND d.deleted_at IS NULL
WHERE u.id = $1 AND u.deleted_at IS NULL;

-- name: GetDoctorSpecialtyIDsForMe :many
SELECT specialty_id
FROM doctors_profile_specialties
WHERE doctor_profile_id = $1;