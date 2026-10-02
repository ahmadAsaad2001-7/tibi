package eta

import (
	"context"

	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
	"tibi/internal/queue/eta/db"
)

const (
	floorMinutes = 20
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

// AverageMinutes returns the rolling average duration of completed entries
// in this window. Falls back to the booked duration, then to 20.
func (s *Service) AverageMinutes(ctx context.Context, windowID int64, bookedDuration int) (int, error) {
	q := db.New(s.db.Querier(ctx))
	row, err := q.AverageCompletedSeconds(ctx, windowID)
	if err != nil {
		return 0, httpx.Internal(err)
	}
	if row > 0 {
		m := int(row / 60)
		if m > 0 {
			return m, nil
		}
	}
	if bookedDuration > 0 {
		return bookedDuration, nil
	}
	return floorMinutes, nil
}
