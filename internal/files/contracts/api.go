package contracts

import "context"

// API is the only surface other modules may import from Files.
type API interface {
	// Get returns metadata for a live file. Returns ErrNotFound if the
	// file is missing or soft-deleted.
	Get(ctx context.Context, fileID int64) (*FileMetadata, error)

	// PresignGet returns a time-limited URL for the file's bytes.
	// Callers are responsible for authorization before calling this.
	PresignGet(ctx context.Context, fileID int64, ttlSeconds int) (string, error)
}

type FileMetadata struct {
	ID           int64
	UploaderID   int64
	Scope        string
	OriginalName string
	ContentType  string
	SizeBytes    int64
}

var ErrNotFound = errNotFound{}

type errNotFound struct{}

func (errNotFound) Error() string { return "file not found" }
