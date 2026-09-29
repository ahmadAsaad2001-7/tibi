package createprofile

import (
	"context"

	"github.com/ahmadAsaad2001-7/tibi/internal/patients/contracts"
	"github.com/ahmadAsaad2001-7/tibi/internal/patients/createprofile/db"
	"github.com/ahmadAsaad2001-7/tibi/internal/platform/database"
	"github.com/ahmadAsaad2001-7/tibi/internal/platform/httpx"
)

// Service implements contracts.API. It is constructed in main and
// injected into the Identity register service.
type Service struct {
	db *database.DB
}

func NewService(db *database.DB) *Service { return &Service{db: db} }

var _ contracts.API = (*Service)(nil)

func (s *Service) CreatePatientProfile(ctx context.Context, in contracts.CreatePatientProfileInput) (int64, error) {
	q := db.New(s.db.Querier(ctx))
	id, err := q.InsertPatientProfile(ctx, db.InsertPatientProfileParams{
		UserID:      in.UserID,
		FullName:    in.FullName,
		PhoneNumber: in.PhoneNumber,
	})
	if err != nil {
		return 0, httpx.Internal(err)
	}
	return id, nil
}
