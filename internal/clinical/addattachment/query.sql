-- name: GetRecordForAttachment :one
SELECT id, doctor_profile_id FROM clinical_medical_records
WHERE consultation_id = $1 AND deleted_at IS NULL;

-- name: InsertAttachment :one
INSERT INTO clinical_medical_attachments (medical_record_id, file_id, label)
VALUES ($1, $2, $3)
    RETURNING id, created_at;