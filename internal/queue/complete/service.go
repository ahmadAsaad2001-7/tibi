package complete

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	consultationscontracts "tibi/internal/consultations/contracts"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
	"tibi/internal/platform/ws"
	"tibi/internal/queue/complete/db"
	"tibi/internal/queue/getsessionqueue"
	"tibi/internal/queue/notify"
	"tibi/internal/queue/queueentry"
)

type Service struct {
	db            *database.DB
	consultations consultationscontracts.API
	hub           ws.Hub
	snapshot      *getsessionqueue.Service
	clock         func() time.Time
}

func NewService(db *database.DB, c consultationscontracts.API, hub ws.Hub, snapshot *getsessionqueue.Service) *Service {
	return &Service{db: db, consultations: c, hub: hub, snapshot: snapshot, clock: time.Now}
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

		entry := hydrateEntry(row)
		now := s.clock()
		if err := entry.Complete(now); err != nil {
			if errors.Is(err, queueentry.ErrInvalidTransition) {
				return httpx.Conflict("entry cannot be completed from status " + string(entry.Status))
			}
			return httpx.Internal(err)
		}

		if err := s.persist(ctx, q, entry, row.Xmin); err != nil {
			return err
		}

		// If the completed entry is the doctor's current patient, clear it.
		// ClearWindowCurrent only clears when current_queue_entry_id matches,
		// so a stale or non-current entry is a safe no-op.
		if err := q.ClearWindowCurrent(ctx, db.ClearWindowCurrentParams{
			ID:                  row.QueueWindowID,
			UpdatedAt:           pgTime(now),
			CurrentQueueEntryID: &entryID,
		}); err != nil {
			return httpx.Internal(err)
		}

		// Cross-module write: mark the consultation Completed (section 10).
		if err := s.consultations.MarkCompleted(ctx, row.ConsultationID); err != nil {
			return err
		}

		sessionID = entry.ClinicSessionID
		patientUserID = row.PatientUserID
		resp = entryToResponse(entry)
		return nil
	})
	if err != nil {
		return nil, err
	}
	notify.Fanout(ctx, s.hub, s.snapshot, sessionID, patientUserID, ws.Event{
		Type:    "QueueEntryStatusChanged",
		Payload: map[string]any{"entry_id": entryID, "status": "Completed"},
	})
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
