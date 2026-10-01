package availability

import (
	"time"

	"tibi/internal/doctors/doctorprofile"
	"tibi/internal/doctors/scheduleexception"
	"tibi/internal/doctors/weeklyschedule"
)

// NextSlot returns the earliest slot at or after `from`, within `horizonDays`.
// Returns zero time and false if none found.
//
// Slices 5 uses horizonDays = 30. Bump if a doctor's next slot is consistently
// farther out than that and callers need to see it.
func NextSlot(
	from time.Time,
	horizonDays int,
	blocks []weeklyschedule.Block,
	exceptions []scheduleexception.Exception,
	bookedByDate map[string][]doctorprofile.TimeOfDay,
) (time.Time, bool) {

	startDate := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	byDate := indexExceptionsByDate(exceptions)

	for offset := 0; offset <= horizonDays; offset++ {
		d := startDate.AddDate(0, 0, offset)
		exs := byDate[d.Format("2006-01-02")]

		booked := bookedByDate[d.Format("2006-01-02")]
		for _, s := range ForDate(d, blocks, exs, booked) {
			slot := time.Date(d.Year(), d.Month(), d.Day(),
				s.Start.Hour, s.Start.Minute, 0, 0, time.UTC)
			if !slot.Before(from) {
				return slot, true
			}
		}
	}
	return time.Time{}, false
}

func indexExceptionsByDate(exs []scheduleexception.Exception) map[string][]scheduleexception.Exception {
	out := map[string][]scheduleexception.Exception{}
	for _, e := range exs {
		k := e.Date.Format("2006-01-02")
		out[k] = append(out[k], e)
	}
	return out
}
