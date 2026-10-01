package api

import (
	"context"

	"tibi/internal/payments/contracts"
	"tibi/internal/payments/createpending"
)

type API struct {
	pending *createpending.Service
}

func New(pending *createpending.Service) *API { return &API{pending: pending} }

var _ contracts.API = (*API)(nil)

func (a *API) CreatePendingPayment(ctx context.Context, in contracts.CreatePendingPaymentInput) (*contracts.CreatePendingPaymentOutput, error) {
	out, err := a.pending.Execute(ctx, createpending.Input{
		PatientProfileID: in.PatientProfileID,
		DoctorProfileID:  in.DoctorProfileID,
		ConsultationID:   in.ConsultationID,
		Amount:           in.Amount,
		Currency:         in.Currency,
		Channel:          in.Channel,
	})
	if err != nil {
		return nil, err
	}
	return &contracts.CreatePendingPaymentOutput{
		PaymentID:   out.PaymentID,
		CheckoutURL: out.CheckoutURL,
	}, nil
}
