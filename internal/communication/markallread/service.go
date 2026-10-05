package markallread

import (
	"context"

	"tibi/internal/communication/markallread/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Response struct {
	MarkedRead int64 `json:"marked_read"`
}

func (s *Service) Execute(ctx context.Context, userID int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	n, err := q.MarkAllRead(ctx, userID)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	return &Response{MarkedRead: n}, nil
}
