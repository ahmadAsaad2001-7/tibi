package consultation

import (
	"errors"
	"time"
)

type Status string

const (
	StatusPending    Status = "Pending"
	StatusConfirmed  Status = "Confirmed"
	StatusInProgress Status = "InProgress"
	StatusCompleted  Status = "Completed"
	StatusCancelled  Status = "Cancelled"
	StatusNoShow     Status = "NoShow"
)

type Consultation struct {
	ID               int64
	PatientProfileID int64
	DoctorProfileID  int64
	ClinicSessionID  *int64
	ScheduledAt      time.Time
	DurationMinutes  int
	Status           Status
	IsUrgent         bool
	Notes            *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

var (
	ErrNotPending     = errors.New("consultation is not pending")
	ErrNotCancellable = errors.New("consultation is not cancellable")
)

// Confirm is called by Payments when the webhook succeeds.
func (c *Consultation) Confirm(now time.Time) error {
	if c.Status != StatusPending {
		return ErrNotPending
	}
	c.Status = StatusConfirmed
	c.UpdatedAt = now
	return nil
}

// Cancel is allowed for the patient or the doctor before the consultation
// starts. Once InProgress or terminal, it is refused.
func (c *Consultation) Cancel(now time.Time) error {
	switch c.Status {
	case StatusPending, StatusConfirmed:
		c.Status = StatusCancelled
		c.UpdatedAt = now
		return nil
	}
	return ErrNotCancellable
}

// IsSlotOccupied answers whether this consultation blocks the slot.
// Stale Pendings (>15 min old) do not.
func (c *Consultation) IsSlotOccupied(now time.Time) bool {
	switch c.Status {
	case StatusConfirmed, StatusInProgress:
		return true
	case StatusPending:
		return now.Sub(c.CreatedAt) < 15*time.Minute
	}
	return false
}
