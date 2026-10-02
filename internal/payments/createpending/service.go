package createpending

import (
	"context"
	"strconv"

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

	q := db.New(s.db.Querier(ctx))
	checkout := kOut.SessionURL
	row, err := q.InsertPendingPayment(ctx, db.InsertPendingPaymentParams{
		PatientProfileID: in.PatientProfileID,
		DoctorProfileID:  in.DoctorProfileID,
		ConsultationID:   in.ConsultationID,
		Amount:           in.Amount,
		Currency:         in.Currency,
		Channel:          in.Channel,
		CheckoutUrl:      &checkout,
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
