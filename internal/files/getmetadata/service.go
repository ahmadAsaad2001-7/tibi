package getmetadata

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"tibi/internal/files/getmetadata/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

// ErrNotFound is the sentinel the files/api facade maps to
// contracts.ErrNotFound.
var ErrNotFound = errors.New("file not found")

type Service struct {
	db *database.DB
}

func NewService(db *database.DB) *Service { return &Service{db: db} }

// Internal is the full row, including ObjectKey. Not JSON-serializable
// shape-wise; consumed by the files/api facade and by get/delete services.
type Internal struct {
	ID           int64
	UploaderID   int64
	Scope        string
	ObjectKey    string
	OriginalName string
	ContentType  string
	SizeBytes    int64
}

// Response is the JSON-safe shape returned by the HTTP handler.
type Response struct {
	ID           int64  `json:"id"`
	UploaderID   int64  `json:"uploader_id"`
	Scope        string `json:"scope"`
	OriginalName string `json:"original_name"`
	ContentType  string `json:"content_type"`
	SizeBytes    int64  `json:"size_bytes"`
}

// Execute returns the internal model. Used by the facade and by
// get.Service, which needs ObjectKey to presign.
func (s *Service) Execute(ctx context.Context, fileID int64) (*Internal, error) {
	q := db.New(s.db.Querier(ctx))
	row, err := q.GetFileMetadataByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, httpx.Internal(err)
	}
	return &Internal{
		ID:           row.ID,
		UploaderID:   row.UploaderID,
		Scope:        string(row.Scope),
		ObjectKey:    row.ObjectKey,
		OriginalName: row.OriginalName,
		ContentType:  row.ContentType,
		SizeBytes:    row.SizeBytes,
	}, nil
}

// ExecutePublic returns the JSON-safe shape.
func (s *Service) ExecutePublic(ctx context.Context, fileID int64) (*Response, error) {
	m, err := s.Execute(ctx, fileID)
	if err != nil {
		return nil, err
	}
	return &Response{
		ID:           m.ID,
		UploaderID:   m.UploaderID,
		Scope:        m.Scope,
		OriginalName: m.OriginalName,
		ContentType:  m.ContentType,
		SizeBytes:    m.SizeBytes,
	}, nil
}
