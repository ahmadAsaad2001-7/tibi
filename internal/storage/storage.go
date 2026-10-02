package storage

import (
	"context"
	"io"
	"time"
)

// Storage is the interface every object store driver implements.
// The rest of the system talks to this; nothing else knows whether the
// bytes live on the local disk or in S3-compatible storage.
type Storage interface {
	// Put streams an object into storage. Returns the object key.
	// Caller supplies a suggested key; the implementation may prefix it.
	Put(ctx context.Context, key string, body io.Reader, contentType string, size int64) error

	// PresignGet returns a time-limited URL the client can fetch.
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)

	// Delete removes an object. Idempotent.
	Delete(ctx context.Context, key string) error
}

// Sizer is the interface for drivers that can report object size without
// a full download. Used for integrity checks.
type Sizer interface {
	Size(ctx context.Context, key string) (int64, error)
}
