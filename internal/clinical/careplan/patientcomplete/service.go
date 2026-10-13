package patientcomplete

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/clinical/careplan/patientcomplete/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db *database.DB
}

func NewService(db *database.DB) *Service {
	return &Service{db: db}
}

type Command struct {
	ScheduledDate *string `json:"scheduled_date,omitempty"`
	Note          *string `json:"note,omitempty"`
}

type Response struct {
	ID            int64   `json:"id"`
	ProtocolID    int64   `json:"protocol_id"`
	ScheduledDate string  `json:"scheduled_date"`
	MarkedAt      string  `json:"marked_at"`
	MarkedByUser  int64   `json:"marked_by_user"`
	Note          *string `json:"note,omitempty"`
}

func (s *Service) Execute(ctx context.Context, userID, protocolID int64, cmd Command) (*Response, error) {
	today := time.Now()
	scheduledDate := pgtype.Date{Time: today, Valid: true}
	if cmd.ScheduledDate != nil {
		parsed, err := time.Parse("2006-01-02", *cmd.ScheduledDate)
		if err != nil {
			return nil, httpx.ValidationFailed(map[string]string{"scheduled_date": "expected YYYY-MM-DD"})
		}
		scheduledDate = pgtype.Date{Time: parsed, Valid: true}
	}

	q := db.New(s.db.Querier(ctx))

	profileID, err := q.GetPatientProfileIDForUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.Forbidden("patient profile not found")
		}
		return nil, httpx.Internal(err)
	}

	protocol, err := q.GetProtocolDetail(ctx, protocolID, scheduledDate)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("protocol not found")
		}
		return nil, httpx.Internal(err)
	}

	// Patient must own this protocol
	if protocol.PatientProfileID != profileID {
		return nil, httpx.Forbidden("not your protocol")
	}

	// Protocol must not be cancelled
	if protocol.CancelledAt.Valid {
		return nil, &httpx.APIError{Status: 409, Code: "already_cancelled", Message: "protocol is already cancelled"}
	}

	// Check not already completed for the scheduled date (joined in the detail query)
	if protocol.CompletionMarkedAt.Valid {
		return nil, &httpx.APIError{Status: 409, Code: "already_completed", Message: "already completed for this date"}
	}

	row, err := q.PatientComplete(ctx, db.PatientCompleteParams{
		ProtocolID:     protocolID,
		ScheduledDate:  scheduledDate,
		MarkedByUserID: userID,
		Note:           cmd.Note,
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}

	return &Response{
		ID:            row.ID,
		ProtocolID:    row.ProtocolID,
		ScheduledDate: row.ScheduledDate.Time.Format("2006-01-02"),
		MarkedAt:      row.MarkedAt.Time.Format(time.RFC3339),
		MarkedByUser:  row.MarkedByUserID,
		Note:          row.Note,
	}, nil
}
