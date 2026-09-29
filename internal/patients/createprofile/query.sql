-- name: InsertPatientProfile :one
INSERT INTO patients_profiles (user_id, full_name, phone_number)
VALUES ($1, $2, $3)
    RETURNING id;