package book

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/consultations/book/db"
	"tibi/internal/consultations/consultation"
	doctorscontracts "tibi/internal/doctors/contracts"
	paymentscontracts "tibi/internal/payments/contracts"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db       *database.DB
	doctors  doctorscontracts.API
	payments paymentscontracts.API
	clock    func() time.Time
}

func NewService(db *database.DB, doctors doctorscontracts.API, payments paymentscontracts.API) *Service {
	return &Service{db: db, doctors: doctors, payments: payments, clock: time.Now}
}

type Response struct {
	Consultation struct {
		ID              int64     `json:"id"`
		Status          string    `json:"status"`
		ScheduledAt     time.Time `json:"scheduled_at"`
		DurationMinutes int       `json:"duration_minutes"`
		IsUrgent        bool      `json:"is_urgent"`
	} `json:"consultation"`
	Payment struct {
		ID          int64  `json:"id"`
		Amount      string `json:"amount"`
		Currency    string `json:"currency"`
		Status      string `json:"status"`
		CheckoutURL string `json:"checkout_url"`
	} `json:"payment"`
}

func (s *Service) Execute(ctx context.Context, userID int64, cmd Command) (*Response, error) {
	scheduledAt, err := time.Parse(time.RFC3339, cmd.ScheduledAt)
	if err != nil {
		return nil, httpx.ValidationFailed(map[string]string{"scheduled_at": "must be RFC3339"})
	}
	now := s.clock()
	if !scheduledAt.After(now) {
		return nil, httpx.Unprocessable("scheduled_at must be in the future")
	}

	// Cross-module read: validate the doctor and slot.
	slotInfo, err := s.doctors.ValidateSlot(ctx, cmd.DoctorProfileID, scheduledAt)
	if err != nil {
		return nil, err
	}
	feeAmount, feeCurrency, err := s.doctors.ConsultationFee(ctx, cmd.DoctorProfileID)
	if err != nil {
		return nil, err
	}

	var resp *Response
	err = s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		patientID, err := q.GetPatientProfileForUser(ctx, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.Forbidden("only patients may book")
			}
			return httpx.Internal(err)
		}

		// Lazy cleanup of expired Pendings on this slot, then check.
		if err := q.CancelStalePending(ctx, db.CancelStalePendingParams{
			DoctorProfileID: cmd.DoctorProfileID,
			ScheduledAt:     scheduledAt,
		}); err != nil {
			return httpx.Internal(err)
		}
		n, err := q.CountActiveSlot(ctx, db.CountActiveSlotParams{
			DoctorProfileID: cmd.DoctorProfileID,
			ScheduledAt:     scheduledAt,
		})
		if err != nil {
			return httpx.Internal(err)
		}
		if n > 0 {
			return httpx.Conflict("slot is not available")
		}

		sessionID, err := s.doctors.EnsureSession(ctx, cmd.DoctorProfileID, scheduledAt, slotInfo)
		if err != nil {
			return err
		}

		row, err := q.InsertConsultation(ctx, db.InsertConsultationParams{
			PatientProfileID: patientID,
			DoctorProfileID:  cmd.DoctorProfileID,
			ClinicSessionID:  &sessionID,
			ScheduledAt:      scheduledAt,
			DurationMinutes:  int32(slotInfo.SlotDurationMinutes),
			IsUrgent:         cmd.IsUrgent,
			Notes:            cmd.Notes,
		})
		if err != nil {
			return httpx.Internal(err)
		}

		// Cross-module write: create the pending payment in the same tx.
		payOut, err := s.payments.CreatePendingPayment(ctx, paymentscontracts.CreatePendingPaymentInput{
			PatientProfileID: patientID,
			DoctorProfileID:  cmd.DoctorProfileID,
			ConsultationID:   row.ID,
			Amount:           feeAmount,
			Currency:         feeCurrency,
			Channel:          cmd.PaymentChannel,
		})
		if err != nil {
			return err
		}

		resp = &Response{}
		resp.Consultation.ID = row.ID
		resp.Consultation.Status = string(consultation.StatusPending)
		resp.Consultation.ScheduledAt = scheduledAt
		resp.Consultation.DurationMinutes = slotInfo.SlotDurationMinutes
		resp.Consultation.IsUrgent = cmd.IsUrgent
		resp.Payment.ID = payOut.PaymentID
		resp.Payment.Amount = feeAmount
		resp.Payment.Currency = feeCurrency
		resp.Payment.Status = "Pending"
		resp.Payment.CheckoutURL = payOut.CheckoutURL
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
