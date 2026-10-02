package ws

import "context"

// MembershipChecker answers whether a user may join a group. Implemented by
// the modules that own the resources (Consultations, Queue) and wired in
// main. Platform does not import modules; the interface uses primitives.
type MembershipChecker interface {
	IsConsultationMember(ctx context.Context, userID, consultationID int64) (bool, error)
	IsQueueMember(ctx context.Context, userID, clinicSessionID int64) (bool, error)
	OtherConsultationMember(ctx context.Context, userID, consultationID int64) (int64, error)
}
