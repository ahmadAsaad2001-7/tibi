package createpending

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/payments/kashier"

	"tibi/internal/payments/createpending/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db      *database.DB
	kashier *kashier.Client
}

func NewService(db *database.DB, k *kashier.Client) *Service {
	return &Service{db: db, kashier: k}
}

type Input struct {
	PatientProfileID int64
	DoctorProfileID  int64
	ConsultationID   int64
	Amount           string
	Currency         string
	Channel          string
	ReturnURL        string
	CancelURL        string
}

type Output struct {
	PaymentID   int64
	CheckoutURL string
}

// Execute calls Kashier to create a session, then inserts a Pending payment.
func (s *Service) Execute(ctx context.Context, in Input) (*Output, error) {
	// 1. Call Kashier BEFORE opening the DB transaction for the payment insert.
	// The consultation row already exists, so we can use its ID as the merchant reference.
	kOut, err := s.kashier.CreateSession(ctx, kashier.CreateSessionInput{
		Amount:          in.Amount,
		Currency:        in.Currency,
		MerchantOrderID: itoa(in.ConsultationID),
		ReturnURL:       in.ReturnURL,
		CancelURL:       in.CancelURL,
		Description:     "Consultation #" + itoa(in.ConsultationID),
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}

	// ✅ FIX 1: Convert string amount to pgtype.Numeric
	var amount pgtype.Numeric
	if err := amount.Scan(in.Amount); err != nil {
		return nil, httpx.Internal(fmt.Errorf("invalid amount format: %w", err))
	}

	q := db.New(s.db.Querier(ctx))
	row, err := q.InsertPendingPayment(ctx, db.InsertPendingPaymentParams{
		PatientProfileID: in.PatientProfileID,
		DoctorProfileID:  in.DoctorProfileID,
		ConsultationID:   in.ConsultationID,
		Amount:           amount, // ✅ Fixed: was in.Amount (string)
		Currency:         in.Currency,
		Channel:          db.PaymentChannel(in.Channel), // ✅ Fixed: cast string to db.PaymentChannel
		CheckoutUrl: pgtype.Text{ // ✅ Fixed: wrap string in pgtype.Text
			String: kOut.SessionURL,
			Valid:  true,
		},
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}

	return &Output{
		PaymentID:   row.ID,
		CheckoutURL: kOut.SessionURL,
	}, nil
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
