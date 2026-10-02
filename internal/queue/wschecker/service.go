package wschecker

import (
	"context"

	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
	"tibi/internal/queue/wschecker/db"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

func (s *Service) IsQueueMember(ctx context.Context, userID, clinicSessionID int64) (bool, error) {
	q := db.New(s.db.Querier(ctx))
	ok, err := q.IsQueueMember(ctx, db.IsQueueMemberParams{
		UserID:          userID,
		ClinicSessionID: clinicSessionID,
	})
	if err != nil {
		return false, httpx.Internal(err)
	}
	return ok, nil
}
