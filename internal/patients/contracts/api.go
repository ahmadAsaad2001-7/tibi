package contracts

import "context"

// API is the only surface other modules may import from Patients.
type API interface {
	CreatePatientProfile(ctx context.Context, in CreatePatientProfileInput) (int64, error)
}

type CreatePatientProfileInput struct {
	UserID      int64
	FullName    string
	PhoneNumber string
}
