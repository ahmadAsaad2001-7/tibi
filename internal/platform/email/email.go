package email

import "context"

// Sender delivers one email. Implementations may be synchronous or
// asynchronous; callers treat them as synchronous and best-effort.
type Sender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// Message is an optional richer form; simple Send above covers the
// current slices.
type Message struct {
	To       string
	Subject  string
	TextBody string
	HTMLBody string
}
