package markconfirmed

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	commcontracts "tibi/internal/communication/contracts"
	commnotify "tibi/internal/communication/notify"
	"tibi/internal/consultations/consultation"
	"tibi/internal/consultations/markconfirmed/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
	"tibi/internal/platform/ws"
)

type Service struct {
	db    *database.DB
	hub   ws.Hub
	notif commcontracts.API
	clock func() time.Time
}

func NewService(db *database.DB, hub ws.Hub, notif commcontracts.API) *Service {
	return &Service{db: db, hub: hub, notif: notif, clock: time.Now}
}

// Execute is called by the Payments webhook when a payment succeeds.
// Idempotent: re-confirming an already-Confirmed consultation is a no-op.
func (s *Service) Execute(ctx context.Context, consultationID int64) error {
	var (
		changed                     bool
		patientUserID, doctorUserID int64
		status                      string
	)
	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		row, err := q.GetConsultationForConfirm(ctx, consultationID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("consultation not found")
			}
			return httpx.Internal(err)
		}

		c := &consultation.Consultation{
			ID:        row.ID,
			Status:    consultation.Status(row.Status),
			CreatedAt: row.CreatedAt.Time,
			UpdatedAt: row.UpdatedAt.Time,
		}

		if err := c.Confirm(s.clock()); err != nil {
			if errors.Is(err, consultation.ErrNotPending) {
				// Already Confirmed, or in a state where Confirm is not
				// a valid transition. For webhook purposes, treat
				// Confirmed as idempotent no-op; anything else is an error.
				if c.Status == consultation.StatusConfirmed {
					return nil
				}
				return httpx.Conflict("consultation is not in a confirmable state")
			}
			return httpx.Internal(err)
		}

		n, err := q.UpdateConsultationStatus(ctx, db.UpdateConsultationStatusParams{
			ID:        c.ID,
			Status:    db.ConsultationStatus(consultation.StatusConfirmed),
			UpdatedAt: pgTime(c.UpdatedAt),
			Xmin:      row.Xmin,
		})
		if err != nil {
			return httpx.Internal(err)
		}
		if n == 0 {
			return httpx.Conflict("consultation was modified concurrently")
		}
		changed = true
		patientUserID = row.PatientUserID
		doctorUserID = row.DoctorUserID
		status = string(c.Status)

		// Inside tx: notify the patient (idempotent — only on the changed path,
		// so webhook retries on an already-Confirmed consultation stay silent).
		if s.notif != nil {
			if err := commnotify.ConsultationConfirmed(ctx, s.notif, patientUserID, consultationID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if changed && s.hub != nil {
		ev := ws.Event{
			Type: "ConsultationStatusChanged",
			Payload: map[string]any{
				"consultation_id": consultationID,
				"status":          status,
			},
		}
		s.hub.SendToConsultation(consultationID, ev)
		s.hub.SendToUser(patientUserID, ev)
		s.hub.SendToUser(doctorUserID, ev)

		// Durable signal: tell the patient to refetch /notifications.
		s.hub.SendToUser(patientUserID, ws.Event{Type: "NotificationCreated", Payload: nil})
	}
	return nil
}
