package contracts

import "context"

// API is the only surface other modules may import from Identity.
type API interface {
	PromotePendingDoctorToDoctor(ctx context.Context, userID int64) error
	DemoteDoctorToPending(ctx context.Context, userID int64) error
}
