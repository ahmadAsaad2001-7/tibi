package createsuspension

import (
	"context"
	"time"

	"tibi/internal/admin/createsuspension/db"
	"tibi/internal/admin/suspension"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

// Service creates suspensions. Called by castvote when a BanUser vote
// resolves For (SD47). Also callable directly by admin tools.
type Service struct {
	db    *database.DB
	clock func() time.Time
}

func NewService(db *database.DB) *Service {
	return &Service{db: db, clock: time.Now}
}

// Create persists a suspension and revokes the user's refresh tokens so the
// suspension is enforced immediately (SD48). Runs inside the caller's tx.
func (s *Service) Create(
	ctx context.Context,
	userID int64, reason string, days int, voteID *int64,
) (int64, error) {
	sus, err := suspension.New(userID, reason, days, voteID, s.clock())
	if err != nil {
		return 0, httpx.ValidationFailed(map[string]string{"suspension": err.Error()})
	}

	q := db.New(s.db.Querier(ctx))
	row, err := q.InsertSuspension(ctx, db.InsertSuspensionParams{
		UserID: sus.UserID,
		Reason: sus.Reason,
		FromTs: pgTime(sus.FromTS),
		ToTs:   pgTimePtr(sus.ToTS),
		VoteID: sus.VoteID,
	})
	if err != nil {
		return 0, httpx.Internal(err)
	}

	// SD48: kill all current sessions immediately.
	if err := q.RevokeAllRefreshTokens(ctx, userID); err != nil {
		return 0, httpx.Internal(err)
	}

	return row.ID, nil
}