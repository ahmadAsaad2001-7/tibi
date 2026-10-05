package updatepost

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

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

	// ✅ FIX: تحويل *string إلى pgtype.Text
	titleText := pgtype.Text{Valid: false}
	if cmd.Title != nil {
		titleText = pgtype.Text{String: *cmd.Title, Valid: true}
	}

	contentText := pgtype.Text{Valid: false}
	if cmd.Content != nil {
		contentText = pgtype.Text{String: *cmd.Content, Valid: true}
	}

	excerptText := pgtype.Text{Valid: false}
	if cmd.Excerpt != nil {
		excerptText = pgtype.Text{String: *cmd.Excerpt, Valid: true}
	}

	coverImageText := pgtype.Text{Valid: false}
	if cmd.CoverImageURL != nil {
		coverImageText = pgtype.Text{String: *cmd.CoverImageURL, Valid: true}
	}

	// ✅ FIX: تحويل *string إلى db.NullPostType
	nullPostType := db.NullPostType{Valid: false}
	if cmd.Type != nil {
		pt := db.PostType(*cmd.Type)
		nullPostType = db.NullPostType{PostType: pt, Valid: true}
	}

	row, err := q.UpdatePost(ctx, db.UpdatePostParams{
		ID:              postID,
		DoctorProfileID: profileID,
		Title:           titleText,
		Content:         contentText,
		Excerpt:         excerptText,
		Type:            nullPostType,
		CoverImageUrl:   coverImageText, // ✅ الحرف 'u' صغير كما يولدها sqlc
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("post not found")
		}
		return nil, httpx.Internal(err)
	}

	return &Response{
		ID:        row.ID,
		UpdatedAt: row.UpdatedAt.Time, // ✅ استخراج time.Time
	}, nil
}
