package updatepost

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/doctorposts/updatepost/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Command struct {
	Title         *string `json:"title" validate:"omitempty,min=1,max=200"`
	Content       *string `json:"content" validate:"omitempty,min=1"`
	Excerpt       *string `json:"excerpt" validate:"omitempty,max=500"`
	Type          *string `json:"type" validate:"omitempty,oneof=HealthTip PatientEducation ClinicNews Publication"`
	CoverImageURL *string `json:"cover_image_url" validate:"omitempty,url"`
}

type Response struct {
	ID        int64     `json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Service) Execute(ctx context.Context, userID, postID int64, cmd Command) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	profileID, err := q.ProfileID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.Forbidden("doctor profile required")
		}
		return nil, httpx.Internal(err)
	}
	row, err := q.UpdatePost(ctx, db.UpdateParams{
		ID:              postID,
		DoctorProfileID: profileID,
		Title:           cmd.Title,
		Content:         cmd.Content,
		Excerpt:         cmd.Excerpt,
		Type:            cmd.Type,
		CoverImageURL:   cmd.CoverImageURL,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("post not found")
		}
		return nil, httpx.Internal(err)
	}
	return &Response{ID: row.ID, UpdatedAt: row.UpdatedAt}, nil
}
