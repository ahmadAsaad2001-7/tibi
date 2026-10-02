package queueentry

import "time"

type Status string

const (
	StatusWaiting    Status = "Waiting"
	StatusCalled     Status = "Called"
	StatusInProgress Status = "InProgress"
	StatusCompleted  Status = "Completed"
	StatusSkipped    Status = "Skipped"
	StatusNoShow     Status = "NoShow"
	StatusCancelled  Status = "Cancelled"
)

func (s Status) IsActive() bool {
	return s == StatusWaiting || s == StatusCalled || s == StatusSkipped
}

func (s Status) IsTerminal() bool {
	return s == StatusCompleted || s == StatusNoShow || s == StatusCancelled
}

type Entry struct {
	ID               int64
	ConsultationID   int64
	QueueWindowID    int64
	ClinicSessionID  int64
	PatientProfileID int64
	QueueNumber      int
	IsPriority       bool
	Status           Status
	CheckedInAt      time.Time
	CalledAt         *time.Time
	StartedAt        *time.Time
	CompletedAt      *time.Time
	SkipReason       *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (e *Entry) Call(now time.Time) error {
	if e.Status != StatusWaiting && e.Status != StatusSkipped {
		return ErrInvalidTransition
	}
	e.Status = StatusCalled
	e.CalledAt = &now
	e.UpdatedAt = now
	return nil
}

func (e *Entry) Start(now time.Time) error {
	if e.Status != StatusCalled {
		return ErrInvalidTransition
	}
	e.Status = StatusInProgress
	e.StartedAt = &now
	e.UpdatedAt = now
	return nil
}

func (e *Entry) Complete(now time.Time) error {
	if e.Status != StatusInProgress {
		return ErrInvalidTransition
	}
	e.Status = StatusCompleted
	e.CompletedAt = &now
	e.UpdatedAt = now
	return nil
}

func (e *Entry) Skip(reason string, now time.Time) error {
	if e.Status != StatusWaiting && e.Status != StatusCalled {
		return ErrInvalidTransition
	}
	e.Status = StatusSkipped
	e.SkipReason = &reason
	e.UpdatedAt = now
	return nil
}

func (e *Entry) NoShow(reason string, now time.Time) error {
	if e.Status != StatusWaiting && e.Status != StatusCalled && e.Status != StatusSkipped {
		return ErrInvalidTransition
	}
	e.Status = StatusNoShow
	e.SkipReason = &reason
	e.UpdatedAt = now
	return nil
}

func (e *Entry) Cancel(now time.Time) error {
	if e.Status != StatusWaiting && e.Status != StatusCalled {
		return ErrInvalidTransition
	}
	e.Status = StatusCancelled
	e.UpdatedAt = now
	return nil
}
