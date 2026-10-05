package call

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	commcontracts "tibi/internal/communication/contracts"
	commnotify "tibi/internal/communication/notify"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
	"tibi/internal/platform/ws"
	"tibi/internal/queue/call/db"
	"tibi/internal/queue/getsessionqueue"
	"tibi/internal/queue/notify"
	"tibi/internal/queue/queueentry"
)

type Service struct {
	db       *database.DB
	hub      ws.Hub
	snapshot *getsessionqueue.Service
	notif    commcontracts.API
	clock    func() time.Time
}

func NewService(db *database.DB, hub ws.Hub, snapshot *getsessionqueue.Service, notif commcontracts.API) *Service {
	return &Service{db: db, hub: hub, snapshot: snapshot, notif: notif, clock: time.Now}
}

type Response struct {
	ID          int64      `json:"id"`
	Status      string     `json:"status"`
	CalledAt    *time.Time `json:"called_at,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	SkipReason  *string    `json:"skip_reason,omitempty"`
}

func (s *Service) Execute(ctx context.Context, userID, entryID int64) (*Response, error) {
	var resp *Response
	var sessionID, patientUserID int64
	var queueNumber int
	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))
		row, err := q.GetEntryForUpdate(ctx, db.GetEntryForUpdateParams{
			ID:     entryID,
			UserID: userID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("queue entry not found")
			}
			return httpx.Internal(err)
		}

		hasOther, err := q.HasOtherActive(ctx, db.HasOtherActiveParams{
			QueueWindowID: row.QueueWindowID,
			ID:            entryID,
		})
		if err != nil {
			return httpx.Internal(err)
		}
		if hasOther {
			return httpx.Conflict("another patient is already called or in progress")
		}

		entry := hydrateEntry(row)
		now := s.clock()
		if err := entry.Call(now); err != nil {
			if errors.Is(err, queueentry.ErrInvalidTransition) {
				return httpx.Conflict("entry cannot be called from status " + string(entry.Status))
			}
			return httpx.Internal(err)
		}

		if err := s.persist(ctx, q, entry, row.Xmin); err != nil {
			return err
		}
		if err := q.SetWindowCurrent(ctx, db.SetWindowCurrentParams{
			ID:                  row.QueueWindowID,
			CurrentQueueEntryID: &entryID,
			UpdatedAt:           pgTime(now),
		}); err != nil {
			return httpx.Internal(err)
		}

		sessionID = entry.ClinicSessionID
		patientUserID = row.PatientUserID
		queueNumber = entry.QueueNumber
		resp = entryToResponse(entry)

		// Inside tx: create the notification row (commits/rolls back with the
		// call). The hub signal fires after commit.
		if s.notif != nil {
			if err := commnotify.QueueCalled(ctx, s.notif, patientUserID, entryID, queueNumber); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	notify.Fanout(ctx, s.hub, s.snapshot, sessionID, patientUserID,
		ws.Event{Type: "QueueEntryCalled", Payload: map[string]any{"entry_id": entryID, "queue_number": queueNumber}},
		ws.Event{Type: "QueueEntryStatusChanged", Payload: map[string]any{"entry_id": entryID, "status": "Called"}},
	)
	// Durable signal: tell the patient to refetch /notifications.
	if s.hub != nil && patientUserID != 0 {
		s.hub.SendToUser(patientUserID, ws.Event{Type: "NotificationCreated", Payload: nil})
	}
	return resp, nil
}

func (s *Service) persist(ctx context.Context, q *db.Queries, e *queueentry.Entry, xmin string) error {
	n, err := q.UpdateEntryStatus(ctx, db.UpdateEntryStatusParams{
		ID:          e.ID,
		Status:      queueentry.Status(e.Status),
		CalledAt:    tsPtr(e.CalledAt),
		StartedAt:   tsPtr(e.StartedAt),
		CompletedAt: tsPtr(e.CompletedAt),
		SkipReason:  e.SkipReason,
		UpdatedAt:   pgTime(e.UpdatedAt),
		Xmin:        xmin,
	})
	if err != nil {
		return httpx.Internal(err)
	}
	if n == 0 {
		return httpx.Conflict("entry was modified concurrently")
	}
	return nil
}

func hydrateEntry(row db.GetEntryForUpdateRow) *queueentry.Entry {
	return &queueentry.Entry{
		ID:               row.ID,
		ConsultationID:   row.ConsultationID,
		QueueWindowID:    row.QueueWindowID,
		ClinicSessionID:  row.ClinicSessionID,
		PatientProfileID: row.PatientProfileID,
		QueueNumber:      int(row.QueueNumber),
		IsPriority:       row.IsPriority,
		Status:           queueentry.Status(row.Status),
		CheckedInAt:      row.CheckedInAt.Time,
		CalledAt:         pgToTime(row.CalledAt),
		StartedAt:        pgToTime(row.StartedAt),
		CompletedAt:      pgToTime(row.CompletedAt),
		SkipReason:       row.SkipReason,
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}
}

func entryToResponse(e *queueentry.Entry) *Response {
	return &Response{
		ID:          e.ID,
		Status:      string(e.Status),
		CalledAt:    e.CalledAt,
		StartedAt:   e.StartedAt,
		CompletedAt: e.CompletedAt,
		SkipReason:  e.SkipReason,
	}
}
