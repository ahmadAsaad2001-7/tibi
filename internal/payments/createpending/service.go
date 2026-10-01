package createpending

import (
	"context"
	"strconv"

	"tibi/internal/payments/createpending/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Input struct {
	PatientProfileID int64
	DoctorProfileID  int64
	ConsultationID   int64
	Amount           string
	Currency         string
	Channel          string
}

type Output struct {
	PaymentID   int64
	CheckoutURL string
}

// Execute inserts a Pending payment and returns a placeholder checkout URL.
// Slice 7 replaces the placeholder with a real Kashier create-session call.
func (s *Service) Execute(ctx context.Context, in Input) (*Output, error) {
	// Placeholder checkout URL, deterministic from consultations id.
	url := "https://pay.kashier.io/session/placeholder?consultation=" +
		itoa(in.ConsultationID)

	q := db.New(s.db.Querier(ctx))
	row, err := q.InsertPendingPayment(ctx, db.InsertPendingPaymentParams{
		PatientProfileID: in.PatientProfileID,
		DoctorProfileID:  in.DoctorProfileID,
		ConsultationID:   in.ConsultationID,
		Amount:           in.Amount,
		Currency:         in.Currency,
		Channel:          in.Channel,
		CheckoutUrl:      &url,
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}
	return &Output{PaymentID: row.ID, CheckoutURL: url}, nil
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
