package contracts

import "context"

// CheckInInfo is the cross-module read the Queue module needs to check a
// patient in. It is returned by GetForCheckIn and is scoped to a confirmed
// consultation owned by the requesting user.
type CheckInInfo struct {
	ConsultationID   int64
	PatientProfileID int64
	DoctorProfileID  int64
	ClinicSessionID  *int64
	DurationMinutes  int
	IsUrgent         bool
}

// API is the only surface other modules may import from Consultations.
type API interface {
	// Confirm transitions a pending consultation to Confirmed.
	// It is called by Payments when a Kashier webhook succeeds.
	Confirm(ctx context.Context, consultationID int64) error

	// GetForCheckIn returns the check-in read for a confirmed consultation
	// owned by userID. It is called by the Queue module on check-in.
	GetForCheckIn(ctx context.Context, consultationID int64, userID int64) (*CheckInInfo, error)

	// MarkCompleted transitions a consultation to Completed. It is called by
	// the Queue module when a queue entry is completed.
	MarkCompleted(ctx context.Context, consultationID int64) error
}
