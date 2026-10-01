package consultation

import (
	"testing"
	"time"
)

func TestConfirmAndCancel(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	c := &Consultation{Status: StatusPending, CreatedAt: now.Add(-time.Minute)}
	if err := c.Confirm(now); err != nil {
		t.Fatal(err)
	}
	if c.Status != StatusConfirmed {
		t.Fatalf("status = %s", c.Status)
	}
	if err := c.Confirm(now); err != ErrNotPending {
		t.Fatalf("second confirm = %v", err)
	}
	if err := c.Cancel(now); err != nil {
		t.Fatal(err)
	}
	if c.Status != StatusCancelled {
		t.Fatalf("status = %s", c.Status)
	}
	if err := c.Cancel(now); err != ErrNotCancellable {
		t.Fatalf("second cancel = %v", err)
	}
}

func TestPendingOccupiesSlotForFifteenMinutes(t *testing.T) {
	created := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	c := &Consultation{Status: StatusPending, CreatedAt: created}
	if !c.IsSlotOccupied(created.Add(14 * time.Minute)) {
		t.Fatal("fresh pending should occupy the slot")
	}
	if c.IsSlotOccupied(created.Add(15 * time.Minute)) {
		t.Fatal("stale pending should not occupy the slot")
	}
	c.Status = StatusConfirmed
	if !c.IsSlotOccupied(created.Add(time.Hour)) {
		t.Fatal("confirmed should occupy the slot")
	}
	c.Status = StatusCancelled
	if c.IsSlotOccupied(created) {
		t.Fatal("cancelled should not occupy the slot")
	}
}
