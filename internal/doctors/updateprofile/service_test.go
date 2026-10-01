package updateprofile

import "testing"

func TestDedupePreservesOrder(t *testing.T) {
	got := dedupe([]int64{3, 1, 3, 2, 1})
	want := []int64{3, 1, 2}
	if len(got) != len(want) {
		t.Fatalf("dedupe = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("dedupe = %v, want %v", got, want)
		}
	}
}
