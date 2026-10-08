package email

import (
	"context"
	"log/slog"
)

// Logger writes emails to the structured log. Dev-only. Never use in
// production; it silently drops all email.
type Logger struct {
	log *slog.Logger
}

func NewLogger(log *slog.Logger) *Logger {
	return &Logger{log: log}
}

func (l *Logger) Send(ctx context.Context, to, subject, body string) error {
	l.log.Info("email_out",
		"to", to,
		"subject", subject,
		"body_len", len(body),
	)
	return nil
}
