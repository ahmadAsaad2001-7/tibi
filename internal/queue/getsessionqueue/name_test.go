package getsessionqueue

import "testing"

func TestAbbreviateName(t *testing.T) {
	if got := abbreviateName("Sara Ibrahim"); got != "Sara I." {
		t.Fatalf("got %q", got)
	}
	if got := abbreviateName("Sara"); got != "Sara" {
		t.Fatalf("got %q", got)
	}
	if got := abbreviateName(""); got != "" {
		t.Fatalf("got %q", got)
	}
}
