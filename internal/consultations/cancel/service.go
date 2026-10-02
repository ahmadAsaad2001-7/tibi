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
	"tibi/internal/platform/observer"
)

type Service struct {
	db       *database.DB
	observer observer.CancellationObserver
	clock    func() time.Time
}

func NewService(db *database.DB, obs observer.CancellationObserver) *Service {
	return &Service{db: db, observer: obs, clock: time.Now}
}

type Response struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

func (s *Service) Execute(ctx context.Context, userID, id int64) (*Response, error) {
	var resp *Response
	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))
		row, err := q.GetForCancel(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("consultation not found")
			}
			return httpx.Internal(err)
		}
		if userID != row.PatientUserID && userID != row.DoctorUserID {
			return httpx.Forbidden("not a participant")
		}
		now := s.clock()
		if !row.ScheduledAt.After(now) {
			return httpx.Unprocessable("consultation has already started")
		}
		c := &consultation.Consultation{Status: consultation.Status(row.Status)}
		if err := c.Cancel(now); err != nil {
			return httpx.Unprocessable(err.Error())
		}
		n, err := q.UpdateStatus(ctx, id, string(c.Status), now, row.Xmin)
		if err != nil {
			return httpx.Internal(err)
		}
		if n == 0 {
			return httpx.Conflict("consultation was updated concurrently")
		}
		if s.observer != nil {
			if err := s.observer.OnConsultationCancelled(ctx, id); err != nil {
				return err
			}
		}
		resp = &Response{ID: id, Status: string(c.Status)}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
