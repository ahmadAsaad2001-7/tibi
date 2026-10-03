-- name: GetPrescriptionByConsultation :one
SELECT id, consultation_id, patient_profile_id, doctor_profile_id,
       issued_at, created_at, updated_at,
       xmin::text AS xmin
FROM clinical_prescriptions
WHERE consultation_id = $1 AND deleted_at IS NULL;

-- name: InsertPrescription :one
INSERT INTO clinical_prescriptions (
    consultation_id, patient_profile_id, doctor_profile_id
) VALUES ($1, $2, $3)
    RETURNING id, issued_at, created_at, updated_at, xmin::text AS xmin;

-- name: DeleteMedicationsForPrescription :exec
UPDATE clinical_prescribed_medications
SET deleted_at = now(), updated_at = now()
WHERE prescription_id = $1 AND deleted_at IS NULL;

-- name: InsertMedication :one
INSERT INTO clinical_prescribed_medications (
    prescription_id, medication_name, dosage, frequency, duration_days, notes
) VALUES ($1, $2, $3, $4, $5, $6)
    RETURNING id, created_at;

-- name: TouchPrescription :exec
UPDATE clinical_prescriptions SET updated_at = now() WHERE id = $1;