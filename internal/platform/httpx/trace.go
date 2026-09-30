package httpx

import (
	"context"
	"strings"
)

// traceKey is a unique key under which the trace ID is stored on the request
// context. Like auth.ctxKey, it exists only to be a concrete type for
// context.WithValue/Value.
type traceKey struct{}

// WithID stores id on ctx so downstream middleware and handlers can
// retrieve it with ID. If id is empty, ctx is returned unchanged.
func WithID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, traceKey{}, id)
}

// ID returns the trace ID stored on ctx, or "" when none was set.
func ID(ctx context.Context) string {
	id, ok := ctx.Value(traceKey{}).(string)
	if !ok {
		return ""
	}
	return id
}

// FromHeader extracts the trace ID from a W3C "traceparent" header value of
// the form `version-trace-id-parent-id-flags`, e.g. "00-4bf92f3577b34da6
// a3ce929d0e5e4736-00f067aa0ba902b7-01". It returns the 32 hex-char trace-id
// segment, or "" when the header is missing or malformed. The segment's
// 32 characters are validated as hex; the whole value is not otherwise
// parsed so off-spec values degrade to a fresh ID.
func FromHeader(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Split(header, "-")
	if uint(len(parts)) != 4 {
		return ""
	}
	traceID := parts[1]
	if len(traceID) != 32 || !isHex(traceID) {
		return ""
	}
	return traceID
}

func isHex(s string) bool {
	for _, c := range s {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return false
	}
	return true
}
