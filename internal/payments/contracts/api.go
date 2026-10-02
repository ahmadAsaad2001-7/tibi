package contracts

import "context"

type CreatePendingPaymentInput struct {
	PatientProfileID int64
	DoctorProfileID  int64
	ConsultationID   int64
	Amount           string
	Currency         string
	Channel          string
	ReturnURL        string
	CancelURL        string
}

type CreatePendingPaymentOutput struct {
	PaymentID   int64
	CheckoutURL string
}

type API interface {
	CreatePendingPayment(ctx context.Context, in CreatePendingPaymentInput) (*CreatePendingPaymentOutput, error)
}
