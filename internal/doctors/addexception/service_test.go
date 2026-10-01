package addexception

import (
	"context"
	"testing"

	"tibi/internal/platform/httpx"
)

func TestExecuteRejectsBadException(t *testing.T) {
	svc := &Service{}
	_, err := svc.Execute(context.Background(), 1, Command{
		Date: "2026-13-01", FromTime: "09:00", ToTime: "12:00", Type: "Closed",
	})
	if httpx.As(err).Code != "validation_failed" {
		t.Fatalf("bad date err = %v", err)
	}
	_, err = svc.Execute(context.Background(), 1, Command{
		Date: "2026-09-30", FromTime: "12:00", ToTime: "09:00", Type: "Closed",
	})
	if httpx.As(err).Code != "validation_failed" {
		t.Fatalf("reversed times err = %v", err)
	}
	_, err = svc.Execute(context.Background(), 1, Command{
		Date: "2026-09-30", FromTime: "09:00", ToTime: "12:00", Type: "Holiday",
	})
	if httpx.As(err).Code != "validation_failed" {
		t.Fatalf("bad type err = %v", err)
	}
}
