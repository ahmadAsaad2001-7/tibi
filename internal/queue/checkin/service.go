package checkin

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	consultationscontracts "tibi/internal/consultations/contracts"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
	"tibi/internal/platform/ws"
	"tibi/internal/queue/checkin/db"
	"tibi/internal/queue/eta"
	"tibi/internal/queue/getsessionqueue"
	"tibi/internal/queue/notify"
)

type Service struct {
	db            *database.DB
	consultations consultationscontracts.API
	eta           *eta.Service
	hub           ws.Hub
	snapshot      *getsessionqueue.Service
	clock         func() time.Time
}

func NewService(db *database.DB, c consultationscontracts.API, e *eta.Service, hub ws.Hub, snapshot *getsessionqueue.Service) *Service {
	return &Service{db: db, consultations: c, eta: e, hub: hub, snapshot: snapshot, clock: time.Now}
}

type Response struct {
	QueueEntryID         int64  `json:"queue_entry_id"`
	QueueNumber          int    `json:"queue_number"`
	Position             int    `json:"position"`
	EstimatedWaitMinutes int    `json:"estimated_wait_minutes"`
	QueueStatus          string `json:"queue_status"`
	ClinicSessionID      int64  `json:"clinic_session_id"`
}

func (s *Service) Execute(ctx context.Context, userID, consultationID int64) (*Response, error) {
	// Ownership + status check via contract.
	info, err := s.consultations.GetForCheckIn(ctx, consultationID, userID)
	if err != nil {
		return nil, err
	}
	if info.ClinicSessionID == nil {
		return nil, httpx.Unprocessable("consultation has no physical session")
	}
	sessionID := *info.ClinicSessionID

	var resp *Response
	var created bool
	err = s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		// Idempotent check: already checked in?
		existing, err := q.GetExistingEntryForConsultation(ctx, consultationID)
		if err == nil {
			// Replay: compute position/eta and return.
			resp, err = s.buildResponse(ctx, q, existing.ID, int(existing.QueueNumber),
				existing.IsPriority, existing.QueueWindowID, sessionID,
				string(existing.Status), info.DurationMinutes)
			return err
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return httpx.Internal(err)
		}

		// Ensure window exists (opens it if first check-in).
		if _, err := q.EnsureWindow(ctx, sessionID); err != nil {
			return httpx.Internal(err)
		}

		// Lock window: serializes concurrent check-ins for this session.
		win, err := q.GetWindowForUpdate(ctx, sessionID)
		if err != nil {
			return httpx.Internal(err)
		}

		// Allocate queue number atomically.
		assignedNumber, err := q.IncrementQueueNumber(ctx, win.ID)
		if err != nil {
			return httpx.Internal(err)
		}

		// is_priority mirrors the consultation's is_urgent flag (SD4).
		isPriority := info.IsUrgent

		row, err := q.InsertQueueEntry(ctx, db.InsertQueueEntryParams{
			ConsultationID:   consultationID,
			QueueWindowID:    win.ID,
			ClinicSessionID:  sessionID,
			PatientProfileID: info.PatientProfileID,
			QueueNumber:      int32(assignedNumber),
			IsPriority:       isPriority,
		})
		if err != nil {
			// Unique violation = concurrent check-in for the same consultation.
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return httpx.AlreadyExists("consultation is already checked in")
			}
			return httpx.Internal(err)
		}

		resp, err = s.buildResponse(ctx, q, row.ID, int(assignedNumber),
			isPriority, win.ID, sessionID, "Waiting", info.DurationMinutes)
		created = err == nil
		return err
	})
	if err != nil {
		return nil, err
	}
	if created && resp != nil {
		notify.Fanout(ctx, s.hub, s.snapshot, sessionID, userID, ws.Event{
			Type: "QueueEntryAdded",
			Payload: map[string]any{
				"entry_id":     resp.QueueEntryID,
				"queue_number": resp.QueueNumber,
				"status":       "Waiting",
			},
		})
	}
	return resp, nil
}

func (s *Service) buildResponse(ctx context.Context, q *db.Queries,
	entryID int64, queueNumber int, isPriority bool,
	windowID, sessionID int64, status string, bookedDuration int) (*Response, error) {

	ahead, err := q.CountAhead(ctx, db.CountAheadParams{
		QueueWindowID: windowID,
		ID:            entryID,
		IsPriority:    isPriority,
		QueueNumber:   int32(queueNumber),
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}
	position := int(ahead) + 1

	avg, err := s.eta.AverageMinutes(ctx, windowID, bookedDuration)
	if err != nil {
		return nil, err
	}

	return &Response{
		QueueEntryID:         entryID,
		QueueNumber:          queueNumber,
		Position:             position,
		EstimatedWaitMinutes: position * avg,
		QueueStatus:          status,
		ClinicSessionID:      sessionID,
	}, nil
}
