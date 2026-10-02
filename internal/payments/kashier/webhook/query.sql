-- name: PaymentExistsByTransactionID :one
SELECT EXISTS (
    SELECT 1 FROM payments_payments WHERE transaction_id = $1
) AS exists;

-- name: GetPaymentByConsultationForUpdate :one
SELECT id, patient_profile_id, doctor_profile_id, consultation_id,
       amount::text AS amount, currency, channel,
       transaction_id, status, transaction_date, checkout_url,
       created_at, updated_at,
       xmin::text AS xmin
FROM payments_payments
WHERE consultation_id = $1 AND deleted_at IS NULL
    FOR UPDATE;

-- name: UpdatePaymentOutcome :execrows
UPDATE payments_payments
SET status = $2,
    transaction_id = $3,
    transaction_date = $4,
    updated_at = now()
WHERE id = $1 AND xmin::text = $5;