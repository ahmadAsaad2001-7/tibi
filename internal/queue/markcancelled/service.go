package markcancelled

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
	"tibi/internal/platform/observer"
	"tibi/internal/platform/ws"
	"tibi/internal/queue/getsessionqueue"
	"tibi/internal/queue/markcancelled/db"
	"tibi/internal/queue/notify"
)

type Service struct {
	db       *database.DB
	hub      ws.Hub
	snapshot *getsessionqueue.Service
}

func NewService(db *database.DB, hub ws.Hub, snapshot *getsessionqueue.Service) *Service {
	return &Service{db: db, hub: hub, snapshot: snapshot}
}

var _ observer.CancellationObserver = (*Service)(nil)

func (s *Service) OnConsultationCancelled(ctx context.Context, consultationID int64) error {
	var sessionID, patientUserID int64
	var changed bool
	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		row, err := q.GetSessionForCancelledEntry(ctx, consultationID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return httpx.Internal(err)
		}
		sessionID = row.ClinicSessionID
		patientUserID = row.PatientUserID

		n, err := q.CancelQueueEntryByConsultation(ctx, consultationID)
		if err != nil {
			return httpx.Internal(err)
		}
		changed = n > 0
		return nil
	})
	if err != nil {
		return err
	}
	if changed {
		s.db.AfterCommit(ctx, func() {
			notify.Fanout(ctx, s.hub, s.snapshot, sessionID, patientUserID, ws.Event{
				Type:    "QueueEntryStatusChanged",
				Payload: map[string]any{"consultation_id": consultationID, "status": "Cancelled"},
			})
		})
	}
	return nil
}
