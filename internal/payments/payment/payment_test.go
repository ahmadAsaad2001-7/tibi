package payment

import (
	"testing"
	"time"
)

func TestMarkAsSucceeded(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	p := &Payment{Status: StatusPending}
	if err := p.MarkAsSucceeded("txn-1", now); err != nil {
		t.Fatal(err)
	}
	if p.Status != StatusSucceeded || p.TransactionID == nil || *p.TransactionID != "txn-1" {
		t.Fatalf("payment = %+v", p)
	}
	if p.TransactionDate == nil || !p.TransactionDate.Equal(now) {
		t.Fatalf("transaction date = %v", p.TransactionDate)
	}
	if err := p.MarkAsSucceeded("txn-2", now); err != ErrInvalidTransition {
		t.Fatalf("second success = %v", err)
	}
}
