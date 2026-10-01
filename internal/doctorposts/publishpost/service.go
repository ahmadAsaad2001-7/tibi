package publishpost

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/doctorposts/publishpost/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db    *database.DB
	clock func() time.Time
}

func NewService(db *database.DB) *Service {
	return &Service{db: db, clock: time.Now}
}

type Response struct {
	ID          int64     `json:"id"`
	IsPublished bool      `json:"is_published"`
	PublishedAt time.Time `json:"published_at"`
}

func (s *Service) Execute(ctx context.Context, userID, postID int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	profileID, err := q.ProfileID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.Forbidden("doctor profile required")
		}
		return nil, httpx.Internal(err)
	}
	published, err := q.IsPublished(ctx, postID, profileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("post not found")
		}
		return nil, httpx.Internal(err)
	}
	if published {
		return nil, httpx.Conflict("post is already published")
	}
	now := s.clock()
	if err := q.Publish(ctx, postID, profileID, now); err != nil {
		return nil, httpx.Internal(err)
	}
	return &Response{ID: postID, IsPublished: true, PublishedAt: now}, nil
}
