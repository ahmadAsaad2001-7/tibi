package book

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype" // ✅ أضفنا هذا الاستيراد

	commcontracts "tibi/internal/communication/contracts"
	commnotify "tibi/internal/communication/notify"
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
	notif    commcontracts.API
	clock    func() time.Time
}

func NewService(db *database.DB, doctors doctorscontracts.API, payments paymentscontracts.API, notif commcontracts.API) *Service {
	return &Service{db: db, doctors: doctors, payments: payments, notif: notif, clock: time.Now}
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

		// 1. Verify user is a patient
		patientID, err := q.GetPatientProfileForUser(ctx, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.Forbidden("only patients may book")
			}
			return httpx.Internal(err)
		}

		// 2. Ensure session FIRST to get the sessionID
		sessionID, err := s.doctors.EnsureSession(ctx, cmd.DoctorProfileID, scheduledAt, slotInfo)
		if err != nil {
			return err
		}

		// 3. Lock the session row in the DB to prevent concurrent double-bookings
		if _, err := q.LockSession(ctx, sessionID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.Internal(errors.New("session disappeared after EnsureSession"))
			}
			return httpx.Internal(err)
		}

		// ✅ FIX 1: تحويل time.Time إلى pgtype.Timestamptz
		scheduledAtTz := pgtype.Timestamptz{Time: scheduledAt, Valid: true}

		if err := q.CancelStalePending(ctx, db.CancelStalePendingParams{
			DoctorProfileID: cmd.DoctorProfileID,
			ScheduledAt:     scheduledAtTz, // ✅ تم التصحيح
		}); err != nil {
			return httpx.Internal(err)
		}

		count, err := q.CountActiveSlot(ctx, db.CountActiveSlotParams{
			DoctorProfileID: cmd.DoctorProfileID,
			ScheduledAt:     scheduledAtTz, // ✅ تم التصحيح
		})
		if err != nil {
			return httpx.Internal(err)
		}
		if count > 0 {
			return httpx.Conflict("slot is not available")
		}

		// ✅ FIX 2: تحويل *int64 إلى pgtype.Int8
		clinicSessionID := pgtype.Int8{Int64: sessionID, Valid: true}

		// ✅ FIX 3: تحويل *string إلى pgtype.Text
		var notes pgtype.Text
		if cmd.Notes != nil {
			notes = pgtype.Text{String: *cmd.Notes, Valid: true}
		}

		row, err := q.InsertConsultation(ctx, db.InsertConsultationParams{
			PatientProfileID: patientID,
			DoctorProfileID:  cmd.DoctorProfileID,
			ClinicSessionID:  clinicSessionID, // ✅ تم التصحيح
			ScheduledAt:      scheduledAtTz,   // ✅ تم التصحيح
			DurationMinutes:  int32(slotInfo.SlotDurationMinutes),
			IsUrgent:         cmd.IsUrgent,
			Notes:            notes, // ✅ تم التصحيح
		})
		if err != nil {
			return httpx.Internal(err)
		}

		var returnURL, cancelURL string
		if cmd.ReturnURL != "" {
			returnURL = cmd.ReturnURL
		}
		if cmd.CancelURL != "" {
			cancelURL = cmd.CancelURL
		}

		// 7. Cross-module write: create the pending payment in the same tx
		payOut, err := s.payments.CreatePendingPayment(ctx, paymentscontracts.CreatePendingPaymentInput{
			PatientProfileID: patientID,
			DoctorProfileID:  cmd.DoctorProfileID,
			ConsultationID:   row.ID,
			Amount:           feeAmount,
			Currency:         feeCurrency,
			Channel:          cmd.PaymentChannel,
			ReturnURL:        returnURL,
			CancelURL:        cancelURL,
		})
		if err != nil {
			return err
		}

		// 7b. Notify the doctor of the new booking, inside the same tx.
		if s.notif != nil {
			doctorUserID, err := q.GetDoctorUserID(ctx, cmd.DoctorProfileID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return httpx.NotFound("doctor profile not found")
				}
				return httpx.Internal(err)
			}
			if err := commnotify.ConsultationBooked(ctx, s.notif, doctorUserID, row.ID, scheduledAt); err != nil {
				return err
			}
		}

		// 8. Build response
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
