package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func New(db DBTX) *Queries { return &Queries{db: db} }

type Queries struct{ db DBTX }

type InsertPendingPaymentParams struct {
	PatientProfileID int64
	DoctorProfileID  int64
	ConsultationID   int64
	Amount           string
	Currency         string
	Channel          string
	CheckoutUrl      *string
}

type PaymentRow struct {
	ID        int64
	CreatedAt time.Time
}

func (q *Queries) InsertPendingPayment(ctx context.Context, arg InsertPendingPaymentParams) (PaymentRow, error) {
	var row PaymentRow
	err := q.db.QueryRow(ctx, `
		INSERT INTO payments_payments (
		    patient_profile_id, doctor_profile_id, consultation_id,
		    amount, currency, channel, checkout_url
		) VALUES ($1, $2, $3, $4::numeric, $5, $6, $7)
		RETURNING id, created_at`,
		arg.PatientProfileID, arg.DoctorProfileID, arg.ConsultationID,
		arg.Amount, arg.Currency, arg.Channel, arg.CheckoutUrl,
	).Scan(&row.ID, &row.CreatedAt)
	return row, err
}
