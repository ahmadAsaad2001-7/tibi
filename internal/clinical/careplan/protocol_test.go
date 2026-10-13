package careplan

import (
	"testing"
	"time"
)

func d(y, m, dd int) time.Time {
	return time.Date(y, time.Month(m), dd, 12, 0, 0, 0, time.UTC)
}

func TestIsActiveOn(t *testing.T) {
	// Daily Apr 10-17.
	p := &Protocol{Frequency: FrequencyDaily, StartDate: d(2026, 4, 10), EndDate: ptr(d(2026, 4, 17))}
	for _, day := range []int{10, 12, 15, 17} {
		if !p.IsActiveOn(d(2026, 4, day)) {
			t.Fatalf("daily: expected active on Apr %d", day)
		}
	}
	if p.IsActiveOn(d(2026, 4, 9)) || p.IsActiveOn(d(2026, 4, 18)) {
		t.Fatal("daily: expected inactive outside range")
	}

	// EveryOtherDay from Apr 10: 10, 12, 14 active; 11, 13 not.
	eo := &Protocol{Frequency: FrequencyEveryOtherDay, StartDate: d(2026, 4, 10)}
	for _, day := range []int{10, 12, 14} {
		if !eo.IsActiveOn(d(2026, 4, day)) {
			t.Fatalf("every-other-day: expected active on Apr %d", day)
		}
	}
	for _, day := range []int{11, 13, 15} {
		if eo.IsActiveOn(d(2026, 4, day)) {
			t.Fatalf("every-other-day: expected inactive on Apr %d", day)
		}
	}

	// Weekly: same weekday as start (Apr 10 2026 is a Friday).
	w := &Protocol{Frequency: FrequencyWeekly, StartDate: d(2026, 4, 10)}
	if !w.IsActiveOn(d(2026, 4, 17)) {
		t.Fatal("weekly: expected active one week later")
	}
	if w.IsActiveOn(d(2026, 4, 11)) {
		t.Fatal("weekly: expected inactive on different weekday")
	}

	// Once: start date only.
	o := &Protocol{Frequency: FrequencyOnce, StartDate: d(2026, 4, 10)}
	if !o.IsActiveOn(d(2026, 4, 10)) || o.IsActiveOn(d(2026, 4, 11)) {
		t.Fatal("once: expected only start date active")
	}

	// Cancelled: never active.
	c := &Protocol{Frequency: FrequencyDaily, StartDate: d(2026, 4, 10), CancelledAt: ptr(d(2026, 4, 12))}
	if c.IsActiveOn(d(2026, 4, 10)) {
		t.Fatal("cancelled: expected inactive")
	}
}

func TestUpdateAndCancel(t *testing.T) {
	p := &Protocol{Title: "x"}
	if err := p.Update("", nil, nil, time.Now()); err != ErrEmptyTitle {
		t.Fatalf("expected ErrEmptyTitle, got %v", err)
	}
	now := time.Now()
	if err := p.Cancel(now); err != nil {
		t.Fatal(err)
	}
	if err := p.Cancel(now); err != ErrAlreadyCancelled {
		t.Fatalf("expected ErrAlreadyCancelled, got %v", err)
	}
	if err := p.Update("y", nil, nil, now); err != ErrAlreadyCancelled {
		t.Fatalf("expected ErrAlreadyCancelled on update, got %v", err)
	}
}

func TestIsActiveOn_DateOnlyAndUnbounded(t *testing.T) {
	// Time of day is ignored: 23:59 on the start date is still the same day.
	p := &Protocol{Frequency: FrequencyOnce, StartDate: d(2026, 4, 10)}
	if !p.IsActiveOn(time.Date(2026, 4, 10, 23, 59, 59, 999999999, time.UTC)) {
		t.Fatal("once: 23:59 on start date must still be active (date-only)")
	}
	if p.IsActiveOn(time.Date(2026, 4, 9, 23, 59, 59, 0, time.UTC)) {
		t.Fatal("once: day before start must be inactive regardless of time")
	}

	// No end_date => unbounded into the future.
	daily := &Protocol{Frequency: FrequencyDaily, StartDate: d(2026, 4, 10)}
	if !daily.IsActiveOn(d(2030, 1, 1)) {
		t.Fatal("daily without end_date: expected active far in the future")
	}
}

func TestIsActiveOn_EndBoundary(t *testing.T) {
	// EveryOtherDay: end_date clamps the run, but only parity days are active.
	// Apr 10 2026 is day 0, so Apr 14 (offset 4) is active, Apr 15 (offset 5) is not.
	eo := &Protocol{Frequency: FrequencyEveryOtherDay, StartDate: d(2026, 4, 10), EndDate: ptr(d(2026, 4, 15))}
	if !eo.IsActiveOn(d(2026, 4, 14)) {
		t.Fatal("every-other-day: expected active on parity day before end")
	}
	if eo.IsActiveOn(d(2026, 4, 15)) || eo.IsActiveOn(d(2026, 4, 16)) {
		t.Fatal("every-other-day: expected inactive at/after end on non-parity days")
	}

	// Weekly: end_date clamps even when the weekday would match.
	// Apr 10 2026 is a Friday; next Friday is Apr 17, past end Apr 15.
	w := &Protocol{Frequency: FrequencyWeekly, StartDate: d(2026, 4, 10), EndDate: ptr(d(2026, 4, 15))}
	if !w.IsActiveOn(d(2026, 4, 10)) {
		t.Fatal("weekly: expected active on start date")
	}
	if w.IsActiveOn(d(2026, 4, 17)) {
		t.Fatal("weekly: matching weekday past end_date must be inactive")
	}

	// Once: an end_date beyond start never extends the single run.
	o := &Protocol{Frequency: FrequencyOnce, StartDate: d(2026, 4, 10), EndDate: ptr(d(2026, 4, 30))}
	if !o.IsActiveOn(d(2026, 4, 10)) || o.IsActiveOn(d(2026, 4, 11)) {
		t.Fatal("once: only start date may be active regardless of end_date")
	}
}

func TestUpdate_HappyPath(t *testing.T) {
	now := time.Date(2026, 4, 12, 9, 0, 0, 0, time.UTC)
	desc := "take with food"
	rem := time.Date(0, 1, 1, 8, 30, 0, 0, time.UTC)
	p := &Protocol{Title: "old", StartDate: d(2026, 4, 10), CreatedAt: now}
	if err := p.Update("new", &desc, &rem, now); err != nil {
		t.Fatalf("update: %v", err)
	}
	if p.Title != "new" {
		t.Fatalf("title = %q", p.Title)
	}
	if p.Description == nil || *p.Description != "take with food" {
		t.Fatalf("description = %v", p.Description)
	}
	if p.ReminderTime == nil || p.ReminderTime.Format("15:04") != "08:30" {
		t.Fatalf("reminder_time = %v", p.ReminderTime)
	}
	if !p.UpdatedAt.Equal(now) {
		t.Fatalf("updated_at = %v, want %v", p.UpdatedAt, now)
	}
	if p.CancelledAt != nil {
		t.Fatal("update must not cancel the protocol")
	}
	// ErrFrozen and ErrInvalidRange are domain sentinels only. The freeze rule
	// (first completion => cannot edit) is enforced by updateprotocol as 409
	// protocol_frozen, and end>=start by createprotocol as 400 validation_failed;
	// both are asserted in integration/careplan_test.go.
}

func ptr(t time.Time) *time.Time { return &t }
