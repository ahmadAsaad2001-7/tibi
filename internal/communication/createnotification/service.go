package createnotification

import (
	"context"
	"encoding/json"

	"tibi/internal/communication/contracts"
	"tibi/internal/communication/createnotification/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

// Create inserts a notification row. It runs on the caller's transaction when
// one is active (via Querier), so the notification commits/rolls back with the
// caller's business write (SD23).
func (s *Service) Create(ctx context.Context, in contracts.CreateNotificationInput) (int64, error) {
	var payloadJSON []byte
	if in.Payload != nil {
		b, err := json.Marshal(in.Payload)
		if err != nil {
			return 0, httpx.Internal(err)
		}
		payloadJSON = b
	}

	q := db.New(s.db.Querier(ctx))
	row, err := q.InsertNotification(ctx, db.InsertNotificationParams{
		UserID:  in.UserID,
		Type:    in.Type,
		Title:   in.Title,
		Body:    in.Body,
		Payload: payloadJSON,
	})
	if err != nil {
		return 0, httpx.Internal(err)
	}
	return row.ID, nil
}
