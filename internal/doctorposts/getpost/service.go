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

type Specialty struct {
	Name string `json:"name"`
}

type Response struct {
	ID                    int64       `json:"id"`
	Title                 string      `json:"title"`
	Content               string      `json:"content"`
	Excerpt               string      `json:"excerpt"`
	Type                  string      `json:"type"`
	CoverImageURL         *string     `json:"cover_image_url"`
	ViewCount             int         `json:"view_count"`
	LikeCount             int         `json:"like_count"`
	PublishedAt           time.Time   `json:"published_at"`
	DoctorID              int64       `json:"doctor_id"`
	DoctorFullName        string      `json:"doctor_full_name"`
	DoctorProfileImageURL *string     `json:"doctor_profile_image_url"`
	Specialties           []Specialty `json:"specialties"`
}

func (s *Service) Execute(ctx context.Context, id int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	row, err := q.GetAndCountView(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("post not found")
		}
		return nil, httpx.Internal(err)
	}
	specs, err := q.Specialties(ctx, row.DoctorID)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	out := make([]Specialty, len(specs))
	for i, sp := range specs {
		out[i] = Specialty{Name: sp.Name}
	}
	return &Response{
		ID:                    row.ID,
		Title:                 row.Title,
		Content:               row.Content,
		Excerpt:               row.Excerpt,
		Type:                  row.Type,
		CoverImageURL:         row.CoverImageURL,
		ViewCount:             int(row.ViewCount),
		LikeCount:             int(row.LikeCount),
		PublishedAt:           row.PublishedAt,
		DoctorID:              row.DoctorID,
		DoctorFullName:        row.DoctorFullName,
		DoctorProfileImageURL: row.DoctorProfileImageURL,
		Specialties:           out,
	}, nil
}
