// Package suspension is the domain model for user suspensions, shared by
// the propose (SD47), create, and lift slices.
package suspension

import (
	"errors"
	"time"
)

type Suspension struct {
	ID         int64
	UserID     int64
	Reason     string
	FromTS     time.Time
	ToTS       *time.Time // nil = permanent
	LiftedAt   *time.Time
	LiftedBy   *int64
	LiftReason *string
	VoteID     *int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

var (
	ErrAlreadyLifted = errors.New("suspension is already lifted")
	ErrAlreadyActive = errors.New("user already has an active suspension")
)

// New constructs a suspension. days=0 means permanent.
func New(userID int64, reason string, days int, voteID *int64, now time.Time) (*Suspension, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user id")
	}
	if len(reason) == 0 || len(reason) > 500 {
		return nil, errors.New("reason must be 1..500 characters")
	}
	if days < 0 || days > 3650 {
		return nil, errors.New("days must be 0..3650")
	}

	var toTS *time.Time
	if days > 0 {
		t := now.AddDate(0, 0, days)
		toTS = &t
	}

	return &Suspension{
		UserID:    userID,
		Reason:    reason,
		FromTS:    now,
		ToTS:      toTS,
		VoteID:    voteID,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// IsActive reports whether the suspension is currently in effect.
func (s *Suspension) IsActive(now time.Time) bool {
	if s.LiftedAt != nil {
		return false
	}
	if s.ToTS != nil && now.After(*s.ToTS) {
		return false
	}
	return true
}

// Lift ends a suspension early. Requires a reason (SD45).
func (s *Suspension) Lift(byAdminID int64, reason string, now time.Time) error {
	if s.LiftedAt != nil {
		return ErrAlreadyLifted
	}
	if reason == "" || len(reason) > 500 {
		return errors.New("lift reason must be 1..500 characters")
	}
	s.LiftedAt = &now
	s.LiftedBy = &byAdminID
	s.LiftReason = &reason
	s.UpdatedAt = now
	return nil
}