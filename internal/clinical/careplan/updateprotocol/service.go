package updateprotocol

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/clinical/careplan/updateprotocol/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db    *database.DB
	clock func() time.Time
}

func NewService(db *database.DB) *Service {
	return &Service{db: db, clock: time.Now}
}

type Command struct {
	Title        *string `json:"title" validate:"omitempty,min=1,max=200"`
	Description  *string `json:"description" validate:"omitempty,max=1000"`
	ReminderTime *string `json:"reminder_time" validate:"omitempty"`
}

type Response struct {
	ID           int64   `json:"id"`
	Title        string  `json:"title"`
	Description  *string `json:"description"`
	ReminderTime *string `json:"reminder_time"`
	UpdatedAt    string  `json:"updated_at"`
}

func (s *Service) Execute(ctx context.Context, userID, protocolID int64, cmd Command) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	row, err := q.GetProtocolForUpdate(ctx, protocolID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("protocol not found")
		}
		return nil, httpx.Internal(err)
	}
	docID, err := doctorProfileID(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	if row.DoctorProfileID != docID {
		return nil, httpx.Forbidden("not your protocol")
	}
	if row.CancelledAt.Valid {
		return nil, &httpx.APIError{Status: 409, Code: "already_cancelled", Message: "protocol is already cancelled"}
	}
	frozen, err := q.HasCompletions(ctx, protocolID)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	if frozen {
		return nil, &httpx.APIError{Status: 409, Code: "protocol_frozen", Message: "protocol is frozen; at least one completion exists"}
	}

	title := row.Title
	if cmd.Title != nil {
		title = *cmd.Title
	}
	if title == "" {
		return nil, httpx.ValidationFailed(map[string]string{"title": "title is required"})
	}
	desc := row.Description
	if cmd.Description != nil {
		desc = cmd.Description
	}
	reminder := row.ReminderTime
	reminderStr := formatTime(reminder)
	if cmd.ReminderTime != nil {
		if *cmd.ReminderTime == "" {
			var zero pgtype.Time
			reminder = zero
			reminderStr = nil
		} else {
			rt, err := time.Parse("15:04", *cmd.ReminderTime)
			if err != nil {
				return nil, httpx.ValidationFailed(map[string]string{"reminder_time": "expected HH:MM"})
			}
			reminder = pgtype.Time{Microseconds: int64(rt.Hour())*3_600_000_000 + int64(rt.Minute())*60_000_000, Valid: true}
			s := rt.Format("15:04")
			reminderStr = &s
		}
	}

	// OCC: compare against the row's xmin snapshot (text), see query.sql.
	now := s.clock()
	n, err := q.UpdateProtocol(ctx, db.UpdateProtocolParams{
		ID: row.ID, Title: title, Description: desc,
		ReminderTime: reminder,
		UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
		Xmin:         row.Xmin,
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}
	if n == 0 {
		return nil, httpx.Conflict("protocol was modified concurrently")
	}
	return &Response{ID: row.ID, Title: title, Description: desc, ReminderTime: reminderStr, UpdatedAt: now.Format(time.RFC3339)}, nil
}

func formatTime(t pgtype.Time) *string {
	if !t.Valid {
		return nil
	}
	h := int(t.Microseconds / 3_600_000_000)
	m := int((t.Microseconds % 3_600_000_000) / 60_000_000)
	s := time.Date(0, 1, 1, h, m, 0, 0, time.UTC).Format("15:04")
	return &s
}

func doctorProfileID(ctx context.Context, q *db.Queries, userID int64) (int64, error) {
	id, err := q.GetDoctorProfileIDForUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, httpx.Forbidden("doctor profile not found")
		}
		return 0, httpx.Internal(err)
	}
	return id, nil
}
