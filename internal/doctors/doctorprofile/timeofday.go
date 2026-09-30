package doctorprofile

import (
	"fmt"
	"time"
)

// TimeOfDay is a wall-clock time with no date or timezone.
type TimeOfDay struct {
	Hour   int
	Minute int
}

func ParseTimeOfDay(s string) (TimeOfDay, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return TimeOfDay{}, fmt.Errorf("invalid time %q: expected HH:MM", s)
	}
	return TimeOfDay{Hour: t.Hour(), Minute: t.Minute()}, nil
}

func (t TimeOfDay) String() string {
	return fmt.Sprintf("%02d:%02d", t.Hour, t.Minute)
}

// Minutes returns minutes since midnight.
func (t TimeOfDay) Minutes() int { return t.Hour*60 + t.Minute }

func (t TimeOfDay) Before(other TimeOfDay) bool { return t.Minutes() < other.Minutes() }
