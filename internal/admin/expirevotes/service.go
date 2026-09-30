package expirevotes

import (
	"context"
	"log/slog"
	"time"

	"tibi/internal/platform/database"
)

type Service struct {
	db   *database.DB
	tick time.Duration
}

func NewService(db *database.DB) *Service {
	return &Service{db: db, tick: 10 * time.Minute}
}

func (s *Service) Run(ctx context.Context) {
	t := time.NewTicker(s.tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := s.expire(ctx); err != nil {
				slog.Error("expire_votes_failed", "err", err)
			}
		}
	}
}

func (s *Service) expire(ctx context.Context) error {
	q := s.db.Querier(ctx)
	tag, err := q.Exec(ctx, `
		UPDATE admin_votes
		SET status = 'Expired', resolved_at = now(), updated_at = now()
		WHERE status = 'Open'
		  AND expires_at < now()
		  AND deleted_at IS NULL`)
	if err != nil {
		return err
	}
	if n := tag.RowsAffected(); n > 0 {
		slog.Info("votes_expired", "count", n)
	}
	return nil
}
