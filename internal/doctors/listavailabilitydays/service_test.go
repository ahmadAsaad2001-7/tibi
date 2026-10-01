package listavailabilitydays

import (
	"context"
	"testing"
	"time"

	"tibi/internal/platform/httpx"
)

func TestExecuteRejectsBadRange(t *testing.T) {
	svc := &Service{}
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.Execute(context.Background(), 1, from.AddDate(0, 0, 1), from)
	if httpx.As(err).Code != "validation_failed" {
		t.Fatalf("reversed range err = %v", err)
	}
	_, err = svc.Execute(context.Background(), 1, from, from.AddDate(0, 0, 61))
	if httpx.As(err).Code != "validation_failed" {
		t.Fatalf("long range err = %v", err)
	}
}
