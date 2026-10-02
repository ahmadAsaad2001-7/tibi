package ws

import "testing"

func TestNoopHubDoesNotPanic(t *testing.T) {
	h := NewNoop()
	h.SendToUser(1, Event{Type: "x"})
	h.SendToConsultation(2, Event{Type: "x"})
	h.SendToQueue(3, Event{Type: "x"})
}

func TestSendToUserCopiesTargets(t *testing.T) {
	h := NewHub(nil, nil).(*hub)
	c := &Client{h: h, userID: 9, send: make(chan Event, 2), groups: map[string]struct{}{}}
	h.register(c)
	h.SendToUser(9, Event{Type: "hello"})
	select {
	case e := <-c.send:
		if e.Type != "hello" {
			t.Fatalf("got %+v", e)
		}
	default:
		t.Fatal("expected event")
	}
}

func TestEnqueueDropsWhenBufferFull(t *testing.T) {
	c := &Client{send: make(chan Event, 1), groups: map[string]struct{}{}}
	c.enqueue(Event{Type: "a"})
	c.enqueue(Event{Type: "b"}) // should not block
	if len(c.send) != 1 {
		t.Fatalf("len = %d", len(c.send))
	}
}
