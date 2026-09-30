package availability

import (
	"time"

	"tibi/internal/doctors/doctorprofile"
	"tibi/internal/doctors/scheduleexception"
	"tibi/internal/doctors/weeklyschedule"
)

type Slot struct {
	Start    doctorprofile.TimeOfDay
	Duration time.Duration
}

// ForDate computes the open slots on a given date, given the doctor's
// active weekly blocks for that day-of-week and any exceptions on that date.
//
// Slice 4 does not subtract existing bookings. The signature takes no
// bookings parameter; slice 5 will extend Input when Consultations exists.
func ForDate(date time.Time, blocks []weeklyschedule.Block, exceptions []scheduleexception.Exception) []Slot {
	dayOfWeek := int(date.Weekday())

	var slots []Slot
	for _, b := range blocks {
		if !b.IsActive || b.DayOfWeek != dayOfWeek {
			continue
		}
		dur := time.Duration(b.SlotDurationMinutes) * time.Minute

		for cursor := b.StartTime; cursor.Minutes()+b.SlotDurationMinutes <= b.EndTime.Minutes(); {
			slotEnd := addMinutes(cursor, b.SlotDurationMinutes)

			blocked := false
			for i := range exceptions {
				if exceptions[i].Blocks(cursor, slotEnd) {
					blocked = true
					break
				}
			}
			if !blocked {
				slots = append(slots, Slot{Start: cursor, Duration: dur})
			}

			cursor = slotEnd
		}
	}
	return slots
}

func addMinutes(t doctorprofile.TimeOfDay, minutes int) doctorprofile.TimeOfDay {
	total := t.Minutes() + minutes
	return doctorprofile.TimeOfDay{Hour: total / 60, Minute: total % 60}
}
