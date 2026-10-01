package doctorprofile

import "testing"

func TestParseTimeOfDay(t *testing.T) {
	got, err := ParseTimeOfDay("09:30")
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "09:30" || got.Minutes() != 9*60+30 {
		t.Fatalf("time = %+v", got)
	}
	if _, err := ParseTimeOfDay("25:00"); err == nil {
		t.Fatal("invalid clock time was accepted")
	}
	short, err := ParseTimeOfDay("9:30")
	if err != nil || short.String() != "09:30" {
		t.Fatalf("short form = %+v err=%v", short, err)
	}
}

func TestTimeOfDayBefore(t *testing.T) {
	early, _ := ParseTimeOfDay("08:00")
	late, _ := ParseTimeOfDay("08:01")
	if !early.Before(late) || late.Before(early) || early.Before(early) {
		t.Fatal("Before comparison is wrong")
	}
}
