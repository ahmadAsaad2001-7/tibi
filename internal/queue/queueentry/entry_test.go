package queueentry

import (
	"testing"
	"time"
)

func TestTransitions(t *testing.T) {
	now := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	e := &Entry{Status: StatusWaiting}
	if err := e.Call(now); err != nil {
		t.Fatal(err)
	}
	if e.Status != StatusCalled || e.CalledAt == nil {
		t.Fatalf("call = %+v", e)
	}
	if err := e.Start(now); err != nil {
		t.Fatal(err)
	}
	if e.Status != StatusInProgress {
		t.Fatalf("start = %s", e.Status)
	}
	if err := e.Complete(now); err != nil {
		t.Fatal(err)
	}
	if e.Status != StatusCompleted {
		t.Fatalf("complete = %s", e.Status)
	}
	if err := e.Call(now); err != ErrInvalidTransition {
		t.Fatalf("call after complete = %v", err)
	}
}

func TestSkipAndRecall(t *testing.T) {
	now := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	e := &Entry{Status: StatusWaiting}
	if err := e.Call(now); err != nil {
		t.Fatal(err)
	}
	if err := e.Skip("not present", now); err != nil {
		t.Fatal(err)
	}
	if e.Status != StatusSkipped || e.SkipReason == nil {
		t.Fatalf("skip = %+v", e)
	}
	if err := e.Call(now); err != nil {
		t.Fatal(err)
	}
	if e.Status != StatusCalled {
		t.Fatalf("recall = %s", e.Status)
	}
}

func TestNoShowFromSkipped(t *testing.T) {
	now := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	e := &Entry{Status: StatusSkipped}
	if err := e.NoShow("left", now); err != nil {
		t.Fatal(err)
	}
	if e.Status != StatusNoShow {
		t.Fatalf("status = %s", e.Status)
	}
}

func TestActiveAndTerminal(t *testing.T) {
	if !StatusWaiting.IsActive() || StatusCompleted.IsActive() {
		t.Fatal("active flags")
	}
	if !StatusNoShow.IsTerminal() || StatusCalled.IsTerminal() {
		t.Fatal("terminal flags")
	}
}
