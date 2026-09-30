package scheduleexception

import (
	"time"

	"tibi/internal/doctors/doctorprofile"
)

type Type string

const (
	TypeClosed        Type = "Closed"
	TypeOpenEarly     Type = "OpenEarly"
	TypeOpenLate      Type = "OpenLate"
	TypeModifiedHours Type = "ModifiedHours"
)

func (t Type) Valid() bool {
	switch t {
	case TypeClosed, TypeOpenEarly, TypeOpenLate, TypeModifiedHours:
		return true
	}
	return false
}

// Exception is a single-day override of the weekly schedule.
// Slice 4 only constructs TypeClosed; the other values exist in the enum
// so the migration does not need to change when they are implemented.
type Exception struct {
	ID              int64
	DoctorProfileID int64
	Date            time.Time // date only; time-of-day components zero
	FromTime        doctorprofile.TimeOfDay
	ToTime          doctorprofile.TimeOfDay
	Type            Type
	Reason          *string
	CreatedAt       time.Time
}

// Blocks reports whether this exception removes the given [start, end)
// slot from the availability. Slice 4: only Closed blocks. When OpenEarly /
// OpenLate / ModifiedHours arrive, this method gains cases.
func (e *Exception) Blocks(slotStart, slotEnd doctorprofile.TimeOfDay) bool {
	if e.Type != TypeClosed {
		return false
	}
	// Overlap if slotStart < e.ToTime AND e.FromTime < slotEnd.
	return slotStart.Minutes() < e.ToTime.Minutes() &&
		e.FromTime.Minutes() < slotEnd.Minutes()
}
