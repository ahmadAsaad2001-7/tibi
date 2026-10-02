package closesession

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
	"tibi/internal/platform/ws"
	"tibi/internal/queue/closesession/db"
	"tibi/internal/queue/getsessionqueue"
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

func (s *Service) Execute(ctx context.Context, userID, sessionID int64) error {
	var closed bool
	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		row, err := q.GetWindowBySessionForUpdate(ctx, db.GetWindowBySessionForUpdateParams{
			ClinicSessionID: sessionID,
			UserID:          userID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("no queue window for this session")
			}
			return httpx.Internal(err)
		}
		if string(row.Status) == "Closed" {
			return nil
		}

		n, err := q.CloseWindow(ctx, db.CloseWindowParams{
			ID:   row.ID,
			Xmin: row.Xmin,
		})
		if err != nil {
			return httpx.Internal(err)
		}
		if n == 0 {
			return httpx.Conflict("window was modified concurrently")
		}

		if _, err := q.FailRemainingEntries(ctx, row.ID); err != nil {
			return httpx.Internal(err)
		}
		closed = true
		return nil
	})
	if err != nil {
		return err
	}
	if closed {
		notify.Fanout(ctx, s.hub, s.snapshot, sessionID, 0, ws.Event{
			Type:    "QueueSessionClosed",
			Payload: map[string]any{"clinic_session_id": sessionID},
		})
	}
	return nil
}
