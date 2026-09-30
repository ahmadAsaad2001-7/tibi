package contracts

import "context"

type API interface {
	CreateDoctorProfile(ctx context.Context, in CreateDoctorProfileInput) (int64, error)
	SetVerificationStatus(ctx context.Context, in SetVerificationStatusInput) error
}

type CreateDoctorProfileInput struct {
	UserID   int64
	FullName string
}

type SetVerificationStatusInput struct {
	UserID          int64
	Status          string // "NotSubmitted" | "PendingReview" | "Verified" | "Rejected"
	RejectionReason *string
}

const (
	RoleDoctor        = "Doctor"
	RolePendingDoctor = "PendingDoctor"
)
