package createpost

import (
	"context"
	"time"

	"tibi/internal/doctorposts/createpost/db"
	doctorscontracts "tibi/internal/doctors/contracts"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db      *database.DB
	doctors doctorscontracts.API
}

func NewService(db *database.DB, doctors doctorscontracts.API) *Service {
	return &Service{db: db, doctors: doctors}
}

type Command struct {
	Title         string  `json:"title" validate:"required,min=1,max=200"`
	Content       string  `json:"content" validate:"required,min=1"`
	Excerpt       *string `json:"excerpt" validate:"omitempty,max=500"`
	Type          string  `json:"type" validate:"required,oneof=HealthTip PatientEducation ClinicNews Publication"`
	CoverImageURL *string `json:"cover_image_url" validate:"omitempty,url"`
}

type Response struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Service) Execute(ctx context.Context, userID int64, cmd Command) (*Response, error) {
	profileID, err := s.doctors.ProfileIDByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	row, err := db.New(s.db.Querier(ctx)).InsertPost(ctx, db.InsertParams{
		DoctorProfileID: profileID,
		Title:           cmd.Title,
		Content:         cmd.Content,
		Excerpt:         cmd.Excerpt,
		Type:            cmd.Type,
		CoverImageURL:   cmd.CoverImageURL,
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}
	return &Response{ID: row.ID, CreatedAt: row.CreatedAt}, nil
}
