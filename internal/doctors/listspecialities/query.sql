-- name: ListSpecialties :many
SELECT id, name, parent_specialty_id
FROM doctors_specialties
ORDER BY name;