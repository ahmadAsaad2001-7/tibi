-- name: PatientProtocols :many
SELECT
    p.id, p.patient_profile_id, p.doctor_profile_id, p.consultation_id,
    p.title, p.description, p.category, p.frequency,
    p.start_date, p.end_date, p.reminder_time,
    p.cancelled_at, p.created_at, p.updated_at, p.deleted_at, p.xmin,
    c.id AS completion_id, c.scheduled_date AS completion_scheduled_date,
    c.marked_at AS completion_marked_at, c.marked_by_user_id AS completion_marked_by_user_id,
    c.note AS completion_note
FROM care_plan_protocols p
LEFT JOIN care_plan_completions c
    ON c.protocol_id = p.id
    AND c.scheduled_date = $2::date
WHERE p.patient_profile_id = $1
  AND p.deleted_at IS NULL
  AND p.cancelled_at IS NULL
ORDER BY p.start_date NULLS LAST, p.id

-- name: GetProtocolDetail :one
SELECT
    p.id, p.patient_profile_id, p.doctor_profile_id, p.consultation_id,
    p.title, p.description, p.category, p.frequency,
    p.start_date, p.end_date, p.reminder_time,
    p.cancelled_at, p.created_at, p.updated_at, p.deleted_at, p.xmin,
    c.id AS completion_id, c.scheduled_date AS completion_scheduled_date,
    c.marked_at AS completion_marked_at, c.marked_by_user_id AS completion_marked_by_user_id,
    c.note AS completion_note
FROM care_plan_protocols p
LEFT JOIN care_plan_completions c
    ON c.protocol_id = p.id
    AND c.scheduled_date = $2::date
WHERE p.id = $1 AND p.deleted_at IS NULL

-- name: IsCompletedToday :one
SELECT EXISTS(
    SELECT 1 FROM care_plan_completions
    WHERE protocol_id = $1 AND scheduled_date = $2::date AND marked_by_user_id = $3
) AS completed

-- name: DoctorPatientProtocols :many
SELECT
    p.id, p.patient_profile_id, p.doctor_profile_id, p.consultation_id,
    p.title, p.description, p.category, p.frequency,
    p.start_date, p.end_date, p.reminder_time,
    p.cancelled_at, p.created_at, p.updated_at, p.deleted_at, p.xmin,
    c.id AS completion_id, c.scheduled_date AS completion_scheduled_date,
    c.marked_at AS completion_marked_at, c.marked_by_user_id AS completion_marked_by_user_id,
    c.note AS completion_note
FROM care_plan_protocols p
LEFT JOIN care_plan_completions c
    ON c.protocol_id = p.id
    AND c.scheduled_date = p.start_date
WHERE p.patient_profile_id = $1
  AND p.deleted_at IS NULL
  AND p.cancelled_at IS NULL
ORDER BY p.start_date NULLS LAST, p.id

-- name: DoctorPatientSummary :one
SELECT
    p.patient_profile_id,
    pp.full_name,
    COUNT(*) FILTER (WHERE p.cancelled_at IS NULL AND p.deleted_at IS NULL) AS active_protocols,
    COUNT(*) FILTER (WHERE c.scheduled_date = sqlc.arg(today)::date) AS today_completed,
    COUNT(*) AS today_total,
    COALESCE(
        (SELECT COUNT(*)::int FROM (
            SELECT gs::date AS d FROM generate_series(
                GREATEST(p.start_date, sqlc.arg(from_date)::date),
                LEAST(COALESCE(p.end_date, sqlc.arg(today)::date), sqlc.arg(today)::date),
                '1 day'
            ) AS gs
            WHERE p.frequency = 'Daily'
               OR (p.frequency = 'EveryOtherDay' AND (gs::date - p.start_date) % 2 = 0)
               OR (p.frequency = 'Weekly' AND EXTRACT(DOW FROM gs) = EXTRACT(DOW FROM p.start_date))
               OR (p.frequency = 'Once' AND gs::date = p.start_date)
        ) t), 0),
        0
    ) AS expected_7d,
    COALESCE(
        (SELECT COUNT(*)::int FROM care_plan_completions c2
         WHERE c2.protocol_id = p.id
           AND c2.scheduled_date BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(today)::date
        ), 0
    ) AS done_7d
FROM care_plan_protocols p
JOIN patients_profiles pp ON pp.id = p.patient_profile_id AND pp.deleted_at IS NULL
LEFT JOIN care_plan_completions c ON c.protocol_id = p.id AND c.scheduled_date = sqlc.arg(today)::date
WHERE p.patient_profile_id = $1
  AND p.deleted_at IS NULL
  AND p.cancelled_at IS NULL
GROUP BY p.id, pp.full_name
HAVING COUNT(*) > 0
