package cancel

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/consultations/cancel/db"
	"tibi/internal/consultations/consultation"
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

type Response struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

func (s *Service) Execute(ctx context.Context, userID, id int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	row, err := q.GetForCancel(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("consultation not found")
		}
		return nil, httpx.Internal(err)
	}
	if userID != row.PatientUserID && userID != row.DoctorUserID {
		return nil, httpx.Forbidden("not a participant")
	}
	now := s.clock()
	if !row.ScheduledAt.After(now) {
		return nil, httpx.Unprocessable("consultation has already started")
	}
	c := &consultation.Consultation{Status: consultation.Status(row.Status)}
	if err := c.Cancel(now); err != nil {
		return nil, httpx.Unprocessable(err.Error())
	}
	n, err := q.UpdateStatus(ctx, id, string(c.Status), now, row.Xmin)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	if n == 0 {
		return nil, httpx.Conflict("consultation was updated concurrently")
	}
	return &Response{ID: id, Status: string(c.Status)}, nil
}
