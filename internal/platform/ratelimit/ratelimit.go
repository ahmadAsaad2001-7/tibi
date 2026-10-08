package ratelimit

import (
	"context"
	"time"
)

// Limiter is the contract every rate limiter implementation satisfies.
// Callers treat errors as fail-open (SD36): log and allow.
type Limiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

// Config describes one limiter.
type Config struct {
	Limit  int           // max events per window
	Window time.Duration // e.g. time.Hour
}

// ClientIP extracts a best-effort client IP from a request. Returns the
// first X-Forwarded-For entry if present, otherwise RemoteAddr.
func ClientIP(remoteAddr, xff string) string {
	if xff != "" {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	// Trim port from RemoteAddr.
	for i := len(remoteAddr) - 1; i >= 0; i-- {
		if remoteAddr[i] == ':' {
			return remoteAddr[:i]
		}
	}
	return remoteAddr
}
