package deleteexception

import (
	"context"

	"tibi/internal/doctors/deleteexception/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

func (s *Service) Execute(ctx context.Context, userID, exceptionID int64) error {
	q := db.New(s.db.Querier(ctx))

	affected, err := q.DeleteException(ctx, db.DeleteExceptionParams{
		ID:     exceptionID,
		UserID: userID,
	})
	if err != nil {
		return httpx.Internal(err)
	}
	if affected == 0 {
		return httpx.NotFound("schedule exception not found")
	}
	return nil
}
