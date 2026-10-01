package weeklyschedule

import (
	"testing"

	"tibi/internal/doctors/doctorprofile"
)

func tod(s string) doctorprofile.TimeOfDay {
	t, err := doctorprofile.ParseTimeOfDay(s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestValidateAcceptsTouchingBlocks(t *testing.T) {
	s := New(1, []Block{
		{DayOfWeek: 1, StartTime: tod("09:00"), EndTime: tod("12:00"), SlotDurationMinutes: 30, IsActive: true},
		{DayOfWeek: 1, StartTime: tod("12:00"), EndTime: tod("17:00"), SlotDurationMinutes: 30, IsActive: true},
		{DayOfWeek: 1, StartTime: tod("18:00"), EndTime: tod("17:00"), SlotDurationMinutes: 30, IsActive: false},
	})
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	active := s.ActiveBlocksFor(1)
	if len(active) != 2 || active[0].StartTime.String() != "09:00" {
		t.Fatalf("active = %+v", active)
	}
	if len(s.ActiveBlocksFor(2)) != 0 {
		t.Fatal("expected no Tuesday blocks")
	}
}

func TestValidateRejectsOverlapAndBounds(t *testing.T) {
	overlap := New(1, []Block{
		{DayOfWeek: 1, StartTime: tod("09:00"), EndTime: tod("12:00"), SlotDurationMinutes: 30, IsActive: true},
		{DayOfWeek: 1, StartTime: tod("11:00"), EndTime: tod("13:00"), SlotDurationMinutes: 30, IsActive: true},
	})
	if err := overlap.Validate(); err == nil {
		t.Fatal("overlapping blocks were accepted")
	}

	backwards := New(1, []Block{
		{DayOfWeek: 3, StartTime: tod("15:00"), EndTime: tod("09:00"), SlotDurationMinutes: 30, IsActive: true},
	})
	if err := backwards.Validate(); err == nil {
		t.Fatal("end before start was accepted")
	}

	badDay := New(1, []Block{
		{DayOfWeek: 8, StartTime: tod("09:00"), EndTime: tod("10:00"), SlotDurationMinutes: 30, IsActive: true},
	})
	if err := badDay.Validate(); err == nil {
		t.Fatal("day 8 was accepted")
	}

	badSlot := New(1, []Block{
		{DayOfWeek: 1, StartTime: tod("09:00"), EndTime: tod("10:00"), SlotDurationMinutes: 4, IsActive: true},
	})
	if err := badSlot.Validate(); err == nil {
		t.Fatal("4 minute slots were accepted")
	}
}
