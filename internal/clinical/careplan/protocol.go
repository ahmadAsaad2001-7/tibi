package careplan

import (
	"errors"
	"time"
)

type Category string

const (
	CategoryMedication Category = "Medication"
	CategoryVitals     Category = "Vitals"
	CategoryLifestyle  Category = "Lifestyle"
	CategoryFollowUp   Category = "FollowUp"
	CategoryOther      Category = "Other"
)

type Frequency string

const (
	FrequencyDaily         Frequency = "Daily"
	FrequencyEveryOtherDay Frequency = "EveryOtherDay"
	FrequencyWeekly        Frequency = "Weekly"
	FrequencyOnce          Frequency = "Once"
)

type Protocol struct {
	ID               int64
	PatientProfileID int64
	DoctorProfileID  int64
	ConsultationID   *int64
	Title            string
	Description      *string
	Category         Category
	Frequency        Frequency
	StartDate        time.Time // date-only
	EndDate          *time.Time
	ReminderTime     *time.Time // time-only (stored as time)
	CancelledAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

var (
	ErrFrozen           = errors.New("protocol is frozen; at least one completion exists")
	ErrAlreadyCancelled = errors.New("protocol is already cancelled")
	ErrInvalidRange     = errors.New("end_date must be >= start_date")
	ErrEmptyTitle       = errors.New("title is required")
)

func (p *Protocol) Update(title string, description *string, reminderTime *time.Time, now time.Time) error {
	if p.CancelledAt != nil {
		return ErrAlreadyCancelled
	}
	if title == "" {
		return ErrEmptyTitle
	}
	p.Title = title
	p.Description = description
	p.ReminderTime = reminderTime
	p.UpdatedAt = now
	return nil
}

func (p *Protocol) Cancel(now time.Time) error {
	if p.CancelledAt != nil {
		return ErrAlreadyCancelled
	}
	p.CancelledAt = &now
	p.UpdatedAt = now
	return nil
}

// IsActiveOn reports whether this protocol schedules a task on the given
// date (date-only, in clinic-local time).
func (p *Protocol) IsActiveOn(date time.Time) bool {
	if p.CancelledAt != nil {
		return false
	}
	d := dateOnly(date)
	if d.Before(dateOnly(p.StartDate)) {
		return false
	}
	if p.EndDate != nil && d.After(dateOnly(*p.EndDate)) {
		return false
	}
	switch p.Frequency {
	case FrequencyDaily:
		return true
	case FrequencyEveryOtherDay:
		return int(d.Sub(dateOnly(p.StartDate)).Hours()/24)%2 == 0
	case FrequencyWeekly:
		return d.Weekday() == dateOnly(p.StartDate).Weekday()
	case FrequencyOnce:
		return d.Equal(dateOnly(p.StartDate))
	}
	return false
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
