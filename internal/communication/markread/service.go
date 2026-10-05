package markread

import (
	"context"

	"tibi/internal/communication/markread/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

// Execute marks a single notification read. Owner-scoped to the caller.
// Already-read, not-owned, or missing all map to a no-op success.
func (s *Service) Execute(ctx context.Context, userID, notificationID int64) error {
	q := db.New(s.db.Querier(ctx))
	_, err := q.MarkNotificationRead(ctx, db.MarkNotificationReadParams{
		ID:     notificationID,
		UserID: userID,
	})
	if err != nil {
		return httpx.Internal(err)
	}
	return nil
}
