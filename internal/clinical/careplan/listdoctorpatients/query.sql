-- name: GetDoctorProfileIDForUser :one
SELECT id FROM doctors_profiles
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: DoctorPatients :many
SELECT DISTINCT p.patient_profile_id AS patient_profile_id,
       pp.full_name AS full_name
FROM care_plan_protocols p
JOIN patients_profiles pp ON pp.id = p.patient_profile_id AND pp.deleted_at IS NULL
WHERE p.doctor_profile_id = $1
  AND p.deleted_at IS NULL
  AND p.cancelled_at IS NULL
ORDER BY pp.full_name;

-- name: ProtocolsActiveOn :many
SELECT
    p.id, p.patient_profile_id, p.doctor_profile_id, p.consultation_id,
    p.title, p.description, p.category, p.frequency,
    p.start_date, p.end_date, p.reminder_time,
    c.marked_at, c.note
FROM care_plan_protocols p
LEFT JOIN care_plan_completions c
    ON c.protocol_id = p.id AND c.scheduled_date = sqlc.arg(scheduled_date)::date
WHERE p.patient_profile_id = sqlc.arg(patient_profile_id)
  AND p.deleted_at IS NULL
  AND p.cancelled_at IS NULL
  AND p.start_date <= sqlc.arg(scheduled_date)::date
  AND (p.end_date IS NULL OR p.end_date >= sqlc.arg(scheduled_date)::date)
  AND (
      p.frequency = 'Daily'
      OR (p.frequency = 'EveryOtherDay'
          AND ((sqlc.arg(scheduled_date)::date - p.start_date) % 2) = 0)
      OR (p.frequency = 'Weekly'
          AND EXTRACT(DOW FROM sqlc.arg(scheduled_date)::date) = EXTRACT(DOW FROM p.start_date))
      OR (p.frequency = 'Once' AND p.start_date = sqlc.arg(scheduled_date)::date)
  )
ORDER BY p.reminder_time NULLS LAST, p.id;

-- name: ProtocolAdherence :one
SELECT
    (
        SELECT COUNT(*)::int FROM (
            SELECT gs::date AS d
            FROM generate_series(
                GREATEST(p.start_date, sqlc.arg(from_date)::date),
                LEAST(COALESCE(p.end_date, sqlc.arg(to_date)::date), sqlc.arg(to_date)::date),
                '1 day'
            ) AS gs
            WHERE
                p.frequency = 'Daily'
                OR (p.frequency = 'EveryOtherDay'
                    AND ((gs::date - p.start_date) % 2) = 0)
                OR (p.frequency = 'Weekly'
                    AND EXTRACT(DOW FROM gs) = EXTRACT(DOW FROM p.start_date))
                OR (p.frequency = 'Once' AND gs::date = p.start_date)
        ) t
    ) AS expected,
    (
        SELECT COUNT(*)::int FROM care_plan_completions c
        WHERE c.protocol_id = p.id
          AND c.scheduled_date BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date
    ) AS done
FROM care_plan_protocols p
WHERE p.id = sqlc.arg(protocol_id);
