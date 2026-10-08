package httpx

import (
	"context"
	"strings"

	"tibi/internal/platform/trace"
)

// WithID stores id on ctx via the shared platform/trace package so that both
// HTTP middleware and the DB tracer read the same value.
func WithID(ctx context.Context, id string) context.Context {
	return trace.WithID(ctx, id)
}

// ID returns the trace ID stored on ctx, or "" when none was set.
func ID(ctx context.Context) string {
	return trace.ID(ctx)
}

// NewTraceID returns a fresh W3C-shaped trace ID.
func NewTraceID() string {
	return trace.NewID()
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
