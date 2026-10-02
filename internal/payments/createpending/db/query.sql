-- name: InsertPendingPayment :one
INSERT INTO payments_payments (
    patient_profile_id, doctor_profile_id, consultation_id,
    amount, currency, channel, checkout_url
) VALUES (
             @patient_profile_id,
             @doctor_profile_id,
             @consultation_id,
             @amount::numeric,       -- Named parameter with cast
             @currency,
             @channel,
             @checkout_url
         )
    RETURNING id, created_at;