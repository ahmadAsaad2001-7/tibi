package api

import (
	"context"
	"tibi/internal/doctors/contracts"
	"tibi/internal/doctors/createprofile/db"
	"tibi/internal/doctors/createsession"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

// API implements contracts.API for the Doctors module.
// This is the only place the Doctors module advertises outward.
type API struct {
	db       *database.DB
	sessions *createsession.Service
}

func New(db *database.DB, sessions *createsession.Service) *API {
	return &API{db: db, sessions: sessions}
}

var _ contracts.API = (*API)(nil)

func (a *API) CreateDoctorProfile(ctx context.Context, in contracts.CreateDoctorProfileInput) (int64, error) {
	q := db.New(a.db.Querier(ctx))
	id, err := q.InsertDoctorProfile(ctx, db.InsertDoctorProfileParams{
		UserID:   in.UserID,
		FullName: in.FullName,
	})
	if err != nil {
		return 0, httpx.Internal(err)
	}
	return id, nil
}

func (a *API) SetVerificationStatus(ctx context.Context, in contracts.SetVerificationStatusInput) error {
	q := db.New(a.db.Querier(ctx))
	affected, err := q.SetDoctorVerificationStatus(ctx, db.SetDoctorVerificationStatusParams{
		UserID:                      in.UserID,
		VerificationStatus:          in.Status,
		VerificationRejectionReason: in.RejectionReason,
	})
	if err != nil {
		return httpx.Internal(err)
	}
	if affected == 0 {
		return httpx.NotFound("doctor profile not found")
	}
	return nil
}
