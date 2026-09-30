package createprofile

import (
	"context"

	"tibi/internal/doctors/contracts"
	"tibi/internal/doctors/createprofile/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db *database.DB
}

func NewService(db *database.DB) *Service { return &Service{db: db} }

func (s *Service) CreateDoctorProfile(ctx context.Context, in contracts.CreateDoctorProfileInput) (int64, error) {
	q := db.New(s.db.Querier(ctx))
	id, err := q.InsertDoctorProfile(ctx, db.InsertDoctorProfileParams{
		UserID:   in.UserID,
		FullName: in.FullName,
	})
	if err != nil {
		return 0, httpx.Internal(err)
	}
	return id, nil
}
