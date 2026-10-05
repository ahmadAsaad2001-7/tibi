package unpublishpost

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype" // ✅ أضفنا هذا الاستيراد

	"tibi/internal/doctorposts/unpublishpost/db"
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
	ID          int64 `json:"id"`
	IsPublished bool  `json:"is_published"`
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

	// ✅ FIX 1: استخدام IsPublishedParams والتقاط القيمتين (bool, error)
	published, err := q.IsPublished(ctx, db.IsPublishedParams{
		ID:              postID,
		DoctorProfileID: profileID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("post not found")
		}
		return nil, httpx.Internal(err)
	}
	if !published {
		return nil, httpx.Conflict("post is not published")
	}

	now := s.clock()

	// ✅ FIX 2: استخدام UnpublishParams والتقاط القيمتين (int64, error) لأن الاستعلام :execrows
	n, err := q.Unpublish(ctx, db.UnpublishParams{
		ID:              postID,
		DoctorProfileID: profileID,
		UpdatedAt:       pgtype.Timestamptz{Time: now, Valid: true}, // ✅ استخدام pgtype.Timestamptz
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}

	// ✅ FIX 3: التحقق من أن الصف تم تحديثه فعلياً
	if n == 0 {
		return nil, httpx.NotFound("post not found or already unpublished")
	}

	return &Response{
		ID:          postID,
		IsPublished: false,
	}, nil
}
