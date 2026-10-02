package getsessionqueue

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
	"tibi/internal/queue/getsessionqueue/db"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type EntryOut struct {
	ID                 int64      `json:"id"`
	QueueNumber        int        `json:"queue_number"`
	Status             string     `json:"status"`
	IsPriority         bool       `json:"is_priority"`
	PatientDisplayName string     `json:"patient_display_name"`
	CheckedInAt        time.Time  `json:"checked_in_at"`
	CalledAt           *time.Time `json:"called_at"`
	StartedAt          *time.Time `json:"started_at"`
	CompletedAt        *time.Time `json:"completed_at"`
}

type Response struct {
	ClinicSessionID int64      `json:"clinic_session_id"`
	Status          string     `json:"status"`
	NextQueueNumber int        `json:"next_queue_number"`
	CurrentEntryID  *int64     `json:"current_entry_id"`
	Entries         []EntryOut `json:"entries"`
}

func (s *Service) Execute(ctx context.Context, userID, sessionID int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	if _, err := q.GetOwnedSession(ctx, db.GetOwnedSessionParams{ID: sessionID, UserID: userID}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("clinic session not found")
		}
		return nil, httpx.Internal(err)
	}
	return s.snapshot(ctx, sessionID)
}

// Snapshot is the same doctor-facing view without an ownership check.
// Used by queue slices after commit to fan out QueueSnapshot events.
func (s *Service) Snapshot(ctx context.Context, sessionID int64) (*Response, error) {
	return s.snapshot(ctx, sessionID)
}

func (s *Service) snapshot(ctx context.Context, sessionID int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	win, winErr := q.GetWindowForSession(ctx, sessionID)
	if winErr != nil && !errors.Is(winErr, pgx.ErrNoRows) {
		return nil, httpx.Internal(winErr)
	}

	rows, err := q.GetSessionQueue(ctx, sessionID)
	if err != nil {
		return nil, httpx.Internal(err)
	}

	resp := &Response{
		ClinicSessionID: sessionID,
		Entries:         make([]EntryOut, len(rows)),
		Status:          "Scheduled",
		NextQueueNumber: 1,
	}
	if winErr == nil {
		resp.Status = string(win.Status)
		resp.NextQueueNumber = int(win.NextQueueNumber)
		resp.CurrentEntryID = win.CurrentQueueEntryID
	}

	for i, r := range rows {
		resp.Entries[i] = EntryOut{
			ID:                 r.ID,
			QueueNumber:        int(r.QueueNumber),
			Status:             string(r.Status),
			IsPriority:         r.IsPriority,
			PatientDisplayName: abbreviateName(r.PatientFullName),
			CheckedInAt:        r.CheckedInAt.Time,
			CalledAt:           pgToTime(r.CalledAt),
			StartedAt:          pgToTime(r.StartedAt),
			CompletedAt:        pgToTime(r.CompletedAt),
		}
	}
	return resp, nil
}

func abbreviateName(full string) string {
	parts := strings.Fields(full)
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	last := parts[len(parts)-1]
	r, _ := utf8.DecodeRuneInString(last)
	if r == utf8.RuneError {
		return parts[0]
	}
	return parts[0] + " " + string(r) + "."
}

func pgToTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	tt := t.Time
	return &tt
}
