-- Cross-module read: joins queue_entries with consultations_consultations,
-- patients_profiles and doctors_profiles for participant ownership (R2).

-- name: GetPatientQueueView :one
SELECT
    qe.id, qe.queue_number, qe.status, qe.is_priority, qe.checked_in_at,
    qe.queue_window_id,
    qw.current_queue_entry_id,
    c_qe.queue_number AS current_queue_number,
    c_qe.status       AS current_queue_status,
    c_qe.started_at   AS current_started_at
FROM queue_entries qe
JOIN queue_windows qw ON qw.id = qe.queue_window_id
LEFT JOIN queue_entries c_qe ON c_qe.id = qw.current_queue_entry_id
JOIN consultations_consultations c ON c.id = qe.consultation_id
JOIN patients_profiles p ON p.id = c.patient_profile_id
JOIN doctors_profiles d ON d.id = c.doctor_profile_id
WHERE qe.consultation_id = $1
  AND qe.status NOT IN ('NoShow', 'Cancelled')
  AND (p.user_id = $2 OR d.user_id = $2);

-- name: CountAhead :one
SELECT COUNT(*)::int AS n
FROM queue_entries
WHERE queue_window_id = sqlc.arg('queue_window_id')
  AND id != sqlc.arg('id')
  AND status IN ('Waiting', 'Called', 'Skipped')
  AND (
      (is_priority = true AND sqlc.arg('is_priority')::boolean = false)
      OR (is_priority = sqlc.arg('is_priority')::boolean AND queue_number < sqlc.arg('queue_number'))
  );
