package markcompleted

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/consultations/markcompleted/db"
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

// Execute is called by the Queue module when a queue entry completes.
// Idempotent: re-completing an already-Completed consultation is a no-op.
func (s *Service) Execute(ctx context.Context, consultationID int64) error {
	return s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		row, err := q.GetConsultationForComplete(ctx, consultationID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("consultation not found")
			}
			return httpx.Internal(err)
		}
		if row.Status == "Completed" || string(row.Status) == "Completed" {
			return nil // idempotent
		}

		now := s.clock()
		n, err := q.UpdateConsultationCompleted(ctx, db.UpdateConsultationCompletedParams{
			ID:        consultationID,
			UpdatedAt: pgTime(now),
			Xmin:      row.Xmin,
		})
		if err != nil {
			return httpx.Internal(err)
		}
		if n == 0 {
			return httpx.Conflict("consultation was modified concurrently")
		}
		return nil
	})
}
