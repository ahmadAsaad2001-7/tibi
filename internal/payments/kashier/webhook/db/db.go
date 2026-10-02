package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

type Queries struct {
	db DBTX
}

const paymentExistsByTransactionID = `-- name: PaymentExistsByTransactionID :one
SELECT EXISTS (
    SELECT 1 FROM payments_payments WHERE transaction_id = $1
) AS exists
`

func (q *Queries) PaymentExistsByTransactionID(ctx context.Context, transactionID *string) (bool, error) {
	row := q.db.QueryRow(ctx, paymentExistsByTransactionID, transactionID)
	var exists bool
	err := row.Scan(&exists)
	return exists, err
}

const getPaymentByConsultationForUpdate = `-- name: GetPaymentByConsultationForUpdate :one
SELECT id, patient_profile_id, doctor_profile_id, consultation_id,
       amount::text AS amount, currency, channel,
       transaction_id, status, transaction_date, checkout_url,
       created_at, updated_at,
       xmin::text AS xmin
FROM payments_payments
WHERE consultation_id = $1 AND deleted_at IS NULL
    FOR UPDATE
`

type GetPaymentByConsultationForUpdateRow struct {
	ID               int64
	PatientProfileID int64
	DoctorProfileID  int64
	ConsultationID   int64
	Amount           string
	Currency         string
	Channel          string
	TransactionId    *string
	Status           string
	TransactionDate  pgtype.Timestamptz
	CheckoutUrl      *string
	CreatedAt        pgtype.Timestamptz
	UpdatedAt        pgtype.Timestamptz
	Xmin             string
}

func (q *Queries) GetPaymentByConsultationForUpdate(ctx context.Context, consultationID int64) (GetPaymentByConsultationForUpdateRow, error) {
	row := q.db.QueryRow(ctx, getPaymentByConsultationForUpdate, consultationID)
	var i GetPaymentByConsultationForUpdateRow
	err := row.Scan(
		&i.ID,
		&i.PatientProfileID,
		&i.DoctorProfileID,
		&i.ConsultationID,
		&i.Amount,
		&i.Currency,
		&i.Channel,
		&i.TransactionId,
		&i.Status,
		&i.TransactionDate,
		&i.CheckoutUrl,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Xmin,
	)
	return i, err
}

const updatePaymentOutcome = `-- name: UpdatePaymentOutcome :execrows
UPDATE payments_payments
SET status = $2,
    transaction_id = $3,
    transaction_date = $4,
    updated_at = now()
WHERE id = $1 AND xmin::text = $5
`

type UpdatePaymentOutcomeParams struct {
	ID              int64
	Status          string
	TransactionID   *string
	TransactionDate pgtype.Timestamptz
	Xmin            string
}

func (q *Queries) UpdatePaymentOutcome(ctx context.Context, arg UpdatePaymentOutcomeParams) (int64, error) {
	tag, err := q.db.Exec(ctx, updatePaymentOutcome,
		arg.ID, arg.Status, arg.TransactionID, arg.TransactionDate, arg.Xmin)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
