package putschedule

import (
	"context"
	"testing"

	"tibi/internal/platform/httpx"
)

func TestExecuteRejectsInvalidClockTimes(t *testing.T) {
	svc := &Service{}
	_, err := svc.Execute(context.Background(), 1, Command{Weekly: []BlockInput{{
		DayOfWeek: 1, StartTime: "9am", EndTime: "10:00", SlotDurationMinutes: 30, IsActive: true,
	}}})
	if httpx.As(err).Code != "validation_failed" {
		t.Fatalf("err = %v", err)
	}
}
