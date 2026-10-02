package cancel

import (
	"context"
	"errors"
	"testing"
)

type stubObserver struct {
	n   int
	err error
}

func (s *stubObserver) OnConsultationCancelled(context.Context, int64) error {
	s.n++
	return s.err
}

func TestNewServiceAcceptsObserver(t *testing.T) {
	obs := &stubObserver{}
	svc := NewService(nil, obs)
	if svc.observer == nil {
		t.Fatal("observer not set")
	}
}

func TestStubObserverError(t *testing.T) {
	obs := &stubObserver{err: errors.New("queue failed")}
	if err := obs.OnConsultationCancelled(context.Background(), 1); err == nil {
		t.Fatal("expected error")
	}
	if obs.n != 1 {
		t.Fatalf("n=%d", obs.n)
	}
}
