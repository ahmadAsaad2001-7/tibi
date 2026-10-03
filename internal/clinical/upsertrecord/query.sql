-- name: GetRecordByConsultation :one
SELECT id, consultation_id, patient_profile_id, doctor_profile_id,
       allergies, current_medications, past_conditions, doctor_notes,
       created_at, updated_at, deleted_at,
       xmin::text AS xmin
FROM clinical_medical_records
WHERE consultation_id = $1 AND deleted_at IS NULL;

-- name: InsertRecord :one
INSERT INTO clinical_medical_records (
    consultation_id, patient_profile_id, doctor_profile_id,
    allergies, current_medications, past_conditions, doctor_notes
) VALUES ($1, $2, $3, $4, $5, $6, $7)
    RETURNING id, created_at, updated_at, xmin::text AS xmin;

-- name: UpdateRecord :execrows
UPDATE clinical_medical_records
SET allergies = $2, current_medications = $3,
    past_conditions = $4, doctor_notes = $5,
    updated_at = $6
WHERE id = $1 AND xmin::text = $7;