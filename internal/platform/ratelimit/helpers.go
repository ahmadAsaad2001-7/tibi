package ratelimit

import (
	"context"
	"log/slog"
)

// AllowOrFailOpen returns true if the limiter allows the key OR if the
// limiter errored. Errors are logged at WARN (SD36).
func AllowOrFailOpen(ctx context.Context, l Limiter, key string) bool {
	ok, err := l.Allow(ctx, key)
	if err != nil {
		slog.Warn("rate limiter error; failing open", "err", err, "key", key)
		return true
	}
	return ok
}