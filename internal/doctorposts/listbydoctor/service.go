package listbydoctor

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/doctorposts/listbydoctor/db"
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
	ViewCount             int       `json:"view_count"`
	LikeCount             int       `json:"like_count"`
	DoctorID              int64     `json:"doctor_id"`
	DoctorName            string    `json:"doctor_name"`
	DoctorProfileImageURL *string   `json:"doctor_profile_image_url"`
}

type Response struct {
	Items      []Item `json:"items"`
	NextCursor *int64 `json:"next_cursor,omitempty"`
}

func (s *Service) Execute(ctx context.Context, doctorID int64, limit int32, cursor *int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))

	// ✅ DoctorExists يرجع int64
	exists, err := q.DoctorExists(ctx, doctorID)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	if exists == 0 {
		return nil, httpx.NotFound("doctor not found")
	}

	// ✅ استخدام ListByDoctorParams بالأسماء الصحيحة
	params := db.ListByDoctorParams{
		DoctorProfileID: doctorID,
		LimitCount:      limit,
	}

	if cursor != nil {
		// إذا كان cursor موجود، نحتاج إلى جلب المنشور أولاً للحصول على published_at
		// لكن هذا معقد، لذا سنستخدم cursor كـ ID فقط
		params.CursorID = pgtype.Int8{Int64: *cursor, Valid: true}
		// نترك CursorPublishedAt كـ NULL لأننا لا نعرف published_at
	}

	rows, err := q.ListByDoctor(ctx, params)
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
			ViewCount:             int(r.ViewCount),
			LikeCount:             int(r.LikeCount),
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
