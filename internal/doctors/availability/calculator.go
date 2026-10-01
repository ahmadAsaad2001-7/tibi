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
// bookedSlots are wall-clock starts already taken. Those times are omitted.
func ForDate(
	date time.Time,
	blocks []weeklyschedule.Block,
	exceptions []scheduleexception.Exception,
	bookedSlots []doctorprofile.TimeOfDay,
) []Slot {

	booked := make(map[int]bool, len(bookedSlots))
	for _, b := range bookedSlots {
		booked[b.Minutes()] = true
	}
	dayOfWeek := int(date.Weekday())

	slots := make([]Slot, 0)
	for _, b := range blocks {
		if !b.IsActive || b.DayOfWeek != dayOfWeek {
			continue
		}
		dur := time.Duration(b.SlotDurationMinutes) * time.Minute

		for cursor := b.StartTime; cursor.Minutes()+b.SlotDurationMinutes <= b.EndTime.Minutes(); cursor = addMinutes(cursor, b.SlotDurationMinutes) {
			if booked[cursor.Minutes()] {
				continue
			}
			slotEnd := addMinutes(cursor, b.SlotDurationMinutes)

			blocked := false
			for i := range exceptions {
				if exceptions[i].Blocks(cursor, slotEnd) {
					blocked = true
					break
				}
			}
			if blocked {
				continue
			}
			slots = append(slots, Slot{Start: cursor, Duration: dur})
		}
	}
	return slots
}

func addMinutes(t doctorprofile.TimeOfDay, minutes int) doctorprofile.TimeOfDay {
	total := t.Minutes() + minutes
	return doctorprofile.TimeOfDay{Hour: total / 60, Minute: total % 60}
}
