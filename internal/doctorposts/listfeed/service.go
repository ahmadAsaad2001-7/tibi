package listfeed

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/doctorposts/listfeed/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Item struct {
	ID                    int64     `json:"id"`
	Title                 string    `json:"title"`
	Excerpt               string    `json:"excerpt"`
	Type                  string    `json:"type"`
	CoverImageURL         *string   `json:"cover_image_url"`
	PublishedAt           time.Time `json:"published_at"`
	DoctorID              int64     `json:"doctor_id"`
	DoctorName            string    `json:"doctor_name"`
	DoctorProfileImageURL *string   `json:"doctor_profile_image_url"`
}

type Response struct {
	Items      []Item `json:"items"`
	NextCursor *int64 `json:"next_cursor,omitempty"`
}

func (s *Service) Execute(ctx context.Context, limit int32, cursor *int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))

	// ✅ استخدام ListFeedParams (الأسماء قد تختلف، تحقق من query.sql.go)
	params := db.ListFeedParams{
		LimitCount: limit,
	}

	if cursor != nil {
		params.CursorID = pgtype.Int8{Int64: *cursor, Valid: true}
	}

	rows, err := q.ListFeed(ctx, params)
	if err != nil {
		return nil, httpx.Internal(err)
	}

	items := make([]Item, len(rows))
	for i, r := range rows {
		var coverImageURL *string
		if r.CoverImageUrl.Valid {
			coverImageURL = &r.CoverImageUrl.String
		}

		var doctorProfileImageURL *string
		if r.DoctorProfileImageUrl.Valid {
			doctorProfileImageURL = &r.DoctorProfileImageUrl.String
		}

		items[i] = Item{
			ID:                    r.ID,
			Title:                 r.Title,
			Excerpt:               r.Excerpt,
			Type:                  string(r.Type),
			CoverImageURL:         coverImageURL,
			PublishedAt:           r.PublishedAt.Time,
			DoctorID:              r.DoctorID,
			DoctorName:            r.DoctorFullName,
			DoctorProfileImageURL: doctorProfileImageURL,
		}
	}

	var nextCursor *int64
	if len(rows) > 0 {
		last := rows[len(rows)-1]
		nextCursor = &last.ID
	}

	return &Response{Items: items, NextCursor: nextCursor}, nil
}
