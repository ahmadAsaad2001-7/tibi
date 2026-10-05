package getpost

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/doctorposts/getpost/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Response struct {
	ID                    int64     `json:"id"`
	Title                 string    `json:"title"`
	Content               string    `json:"content"`
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

func (s *Service) Execute(ctx context.Context, postID int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	row, err := q.GetAndCountView(ctx, postID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("post not found")
		}
		return nil, httpx.Internal(err)
	}

	var coverImageURL *string
	if row.CoverImageUrl.Valid {
		coverImageURL = &row.CoverImageUrl.String
	}

	var doctorProfileImageURL *string
	if row.DoctorProfileImageUrl.Valid {
		doctorProfileImageURL = &row.DoctorProfileImageUrl.String
	}

	return &Response{
		ID:                    row.ID,
		Title:                 row.Title,
		Content:               row.Content,
		Excerpt:               row.Excerpt,
		Type:                  string(row.Type),
		CoverImageURL:         coverImageURL,
		PublishedAt:           row.PublishedAt.Time,
		ViewCount:             int(row.ViewCount),
		LikeCount:             int(row.LikeCount),
		DoctorID:              row.DoctorID,
		DoctorName:            row.DoctorFullName,
		DoctorProfileImageURL: doctorProfileImageURL,
	}, nil
}
