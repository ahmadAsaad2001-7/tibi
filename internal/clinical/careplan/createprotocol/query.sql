-- name: GetDoctorProfileForCarePlan :one
SELECT id, verification_status::text AS verification_status
FROM doctors_profiles
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: PatientExists :one
SELECT id FROM patients_profiles
WHERE id = $1 AND deleted_at IS NULL;

-- name: InsertProtocol :one
INSERT INTO care_plan_protocols (
    patient_profile_id, doctor_profile_id, consultation_id,
    title, description, category, frequency,
    start_date, end_date, reminder_time
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, created_at, updated_at, xmin::text AS xmin;
