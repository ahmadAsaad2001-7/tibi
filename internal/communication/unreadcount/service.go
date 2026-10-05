package unreadcount

import (
	"context"

	"tibi/internal/communication/unreadcount/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Response struct {
	Count int64 `json:"count"`
}

func (s *Service) Execute(ctx context.Context, userID int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	n, err := q.UnreadCount(ctx, userID)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	return &Response{Count: n}, nil
}
