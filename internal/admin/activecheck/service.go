package activecheck

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/admin/activecheck/db"
	"tibi/internal/admin/contracts"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

func (s *Service) ActiveSuspension(ctx context.Context, userID int64) (*contracts.ActiveSuspension, error) {
	q := db.New(s.db.Querier(ctx))
	row, err := q.GetActiveSuspension(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, httpx.Internal(err)
	}
	return &contracts.ActiveSuspension{
		Reason: row.Reason,
		From:   row.FromTs.Time,
		Until:  tsPtr(row.ToTs),
	}, nil
}

func tsPtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	return &ts.Time
}