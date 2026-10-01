package deletepost

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"tibi/internal/doctorposts/deletepost/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

func (s *Service) Execute(ctx context.Context, userID, postID int64) error {
	q := db.New(s.db.Querier(ctx))
	profileID, err := q.ProfileID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.Forbidden("doctor profile required")
		}
		return httpx.Internal(err)
	}
	n, err := q.SoftDelete(ctx, postID, profileID)
	if err != nil {
		return httpx.Internal(err)
	}
	if n == 0 {
		return httpx.NotFound("post not found")
	}
	return nil
}
