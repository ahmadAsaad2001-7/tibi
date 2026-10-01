-- name: InsertPendingPayment :one
INSERT INTO payments_payments (
    patient_profile_id, doctor_profile_id, consultation_id,
    amount, currency, channel, checkout_url
) VALUES ($1, $2, $3, $4::numeric, $5, $6, $7)
    RETURNING id, created_at;

-- name: GetPaymentByConsultation :one
SELECT id, status FROM payments_payments
WHERE consultation_id = $1 AND deleted_at IS NULL;