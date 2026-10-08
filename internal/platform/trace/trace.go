// Package trace holds the request trace ID so HTTP middleware (httpx) and the
// DB tracer (database) can share it without importing each other.
package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type ctxKey struct{}

// WithID stores id on ctx so downstream code can retrieve it with ID.
// If id is empty, ctx is returned unchanged.
func WithID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, id)
}

// ID returns the trace ID stored on ctx, or "" when none was set.
func ID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return ""
}

// NewID returns a 16-byte hex-encoded ID (32 lowercase hex chars), matching
// the W3C Trace Context trace-id segment format.
func NewID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}