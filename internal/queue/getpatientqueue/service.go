package getpatientqueue

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
	"tibi/internal/queue/eta"
	"tibi/internal/queue/getpatientqueue/db"
)

type Service struct {
	db  *database.DB
	eta *eta.Service
}

func NewService(db *database.DB, e *eta.Service) *Service {
	return &Service{db: db, eta: e}
}

type CurrentRef struct {
	QueueNumber int        `json:"queue_number"`
	Status      string     `json:"status"`
	StartedAt   *time.Time `json:"started_at"`
}

type Response struct {
	QueueEntryID         int64       `json:"queue_entry_id"`
	QueueNumber          int         `json:"queue_number"`
	Position             int         `json:"position"`
	Status               string      `json:"status"`
	EstimatedWaitMinutes int         `json:"estimated_wait_minutes"`
	CheckedInAt          time.Time   `json:"checked_in_at"`
	DoctorCurrentlyWith  *CurrentRef `json:"doctor_currently_with"`
}

func (s *Service) Execute(ctx context.Context, userID, consultationID int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	row, err := q.GetPatientQueueView(ctx, db.GetPatientQueueViewParams{
		ConsultationID: consultationID,
		UserID:         userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("no queue entry for this consultation")
		}
		return nil, httpx.Internal(err)
	}

	ahead, err := q.CountAhead(ctx, db.CountAheadParams{
		QueueWindowID: row.QueueWindowID,
		ID:            row.ID,
		IsPriority:    row.IsPriority,
		QueueNumber:   row.QueueNumber,
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}
	position := int(ahead) + 1

	avg, err := s.eta.AverageMinutes(ctx, row.QueueWindowID, 0)
	if err != nil {
		return nil, err
	}

	resp := &Response{
		QueueEntryID:         row.ID,
		QueueNumber:          int(row.QueueNumber),
		Position:             position,
		Status:               string(row.Status),
		EstimatedWaitMinutes: position * avg,
		CheckedInAt:          row.CheckedInAt.Time,
	}
	if row.CurrentQueueEntryID != nil && row.CurrentQueueNumber != nil && row.CurrentQueueStatus.Valid {
		resp.DoctorCurrentlyWith = &CurrentRef{
			QueueNumber: int(*row.CurrentQueueNumber),
			Status:      string(row.CurrentQueueStatus.QueueStatus),
			StartedAt:   pgToTime(row.CurrentStartedAt),
		}
	}
	return resp, nil
}

func pgToTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	tt := t.Time
	return &tt
}
