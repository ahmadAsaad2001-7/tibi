package publishpost

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype" // ✅ أضفنا هذا الاستيراد

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

	// ✅ FIX 1: استخدام IsPublishedParams بدلاً من الوسائط المتعددة
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
	if published {
		return nil, httpx.Conflict("post is already published")
	}

	now := s.clock()

	// ✅ FIX 2: استخدام PublishParams وتغليف الوقت بـ pgtype.Timestamptz
	// ✅ FIX 3: التقاط عدد الصفوف المتأثرة (n) لأن الاستعلام من نوع :execrows
	n, err := q.Publish(ctx, db.PublishParams{
		ID:              postID,
		DoctorProfileID: profileID,
		PublishedAt:     pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:       pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}

	// ✅ FIX 4: التحقق من أن الصف تم تحديثه فعلياً
	if n == 0 {
		return nil, httpx.NotFound("post not found or already published")
	}

	return &Response{
		ID:          postID,
		IsPublished: true,
		PublishedAt: now,
	}, nil
}
