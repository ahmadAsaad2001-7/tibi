package contracts

import (
	"context"
	"time"

	"tibi/internal/doctors/doctorprofile"
)

type API interface {
	CreateDoctorProfile(ctx context.Context, in CreateDoctorProfileInput) (int64, error)
	ProfileIDByUserID(ctx context.Context, userID int64) (int64, error)
	MarkVerified(ctx context.Context, doctorProfileID int64) error
	MarkUnverified(ctx context.Context, doctorProfileID int64) error
	MarkRejected(ctx context.Context, doctorProfileID int64, reason string) error

	// New in slice 6 — booking needs these.
	ValidateSlot(ctx context.Context, doctorProfileID int64, scheduledAt time.Time) (*SlotInfo, error)
	EnsureSession(ctx context.Context, doctorProfileID int64, date time.Time, info *SlotInfo) (int64, error)
	GetDoctorProfileID(ctx context.Context, doctorProfileID int64) (int64, error)
	ConsultationFee(ctx context.Context, doctorProfileID int64) (amount string, currency string, err error)

	// Admin votes still address a doctor by user id.
	SetVerificationStatus(ctx context.Context, in SetVerificationStatusInput) error
}

// SlotInfo describes the weekly block that contains a bookable slot.
type SlotInfo struct {
	BlockStartTime      doctorprofile.TimeOfDay
	BlockEndTime        doctorprofile.TimeOfDay
	SlotDurationMinutes int
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
