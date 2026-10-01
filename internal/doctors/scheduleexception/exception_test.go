package scheduleexception

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

func TestTypeValid(t *testing.T) {
	for _, kind := range []Type{TypeClosed, TypeOpenEarly, TypeOpenLate, TypeModifiedHours} {
		if !kind.Valid() {
			t.Fatalf("%s should be valid", kind)
		}
	}
	if Type("Holiday").Valid() {
		t.Fatal("unknown type reported valid")
	}
}

func TestClosedBlocksOverlapOnly(t *testing.T) {
	closed := &Exception{Type: TypeClosed, FromTime: tod("09:00"), ToTime: tod("12:00")}
	if !closed.Blocks(tod("11:00"), tod("11:30")) {
		t.Fatal("slot inside closed range should be blocked")
	}
	if closed.Blocks(tod("12:00"), tod("12:30")) {
		t.Fatal("slot starting when the closure ends should stay open")
	}
	if closed.Blocks(tod("08:00"), tod("09:00")) {
		t.Fatal("slot ending when the closure starts should stay open")
	}

	early := &Exception{Type: TypeOpenEarly, FromTime: tod("07:00"), ToTime: tod("09:00")}
	if early.Blocks(tod("07:30"), tod("08:00")) {
		t.Fatal("non-closed exceptions do not remove slots yet")
	}
}
