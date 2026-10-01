package availability

import (
	"testing"
	"time"

	"tibi/internal/doctors/doctorprofile"
	"tibi/internal/doctors/scheduleexception"
	"tibi/internal/doctors/weeklyschedule"
)

func tod(s string) doctorprofile.TimeOfDay {
	t, err := doctorprofile.ParseTimeOfDay(s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestForDateSplitsBlocksAndAppliesClosures(t *testing.T) {
	// 2026-09-28 is a Monday. Go's Weekday Sunday is 0.
	monday := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	if monday.Weekday() != time.Monday {
		t.Fatalf("fixture weekday = %s", monday.Weekday())
	}
	blocks := []weeklyschedule.Block{{
		DayOfWeek:           int(time.Monday),
		StartTime:           tod("09:00"),
		EndTime:             tod("10:30"),
		SlotDurationMinutes: 30,
		IsActive:            true,
	}, {
		DayOfWeek:           int(time.Tuesday),
		StartTime:           tod("09:00"),
		EndTime:             tod("10:00"),
		SlotDurationMinutes: 30,
		IsActive:            true,
	}}
	exceptions := []scheduleexception.Exception{{
		Type:     scheduleexception.TypeClosed,
		FromTime: tod("09:00"),
		ToTime:   tod("09:30"),
	}}

	slots := ForDate(monday, blocks, exceptions, nil)
	if len(slots) != 2 {
		t.Fatalf("slots = %+v, want 09:30 and 10:00", slots)
	}
	if slots[0].Start.String() != "09:30" || slots[1].Start.String() != "10:00" {
		t.Fatalf("starts = %s %s", slots[0].Start, slots[1].Start)
	}
	if slots[0].Duration != 30*time.Minute {
		t.Fatalf("duration = %s", slots[0].Duration)
	}

	tuesday := monday.AddDate(0, 0, 1)
	if got := ForDate(tuesday, blocks, nil, nil); len(got) != 2 {
		t.Fatalf("tuesday slots = %+v", got)
	}
	if got := ForDate(monday, blocks, nil, []doctorprofile.TimeOfDay{tod("09:30")}); len(got) != 2 {
		t.Fatalf("booked 09:30 should leave 09:00 and 10:00, got %+v", got)
	}
	if got := ForDate(monday.AddDate(0, 0, 2), blocks, nil, nil); len(got) != 0 {
		t.Fatalf("wednesday should be empty, got %+v", got)
	}
}
