package get

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/files/get/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
	"tibi/internal/storage"
)

type Service struct {
	db      *database.DB
	storage storage.Storage
}

func NewService(db *database.DB, s storage.Storage) *Service {
	return &Service{db: db, storage: s}
}

type Response struct {
	ID           int64     `json:"id"`
	OriginalName string    `json:"original_name"`
	ContentType  string    `json:"content_type"`
	SizeBytes    int64     `json:"size_bytes"`
	FileURL      string    `json:"file_url"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (s *Service) Execute(ctx context.Context, fileID int64, ttl time.Duration) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	row, err := q.GetFileByID(ctx, fileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("file not found")
		}
		return nil, httpx.Internal(err)
	}
	url, err := s.storage.PresignGet(ctx, row.ObjectKey, ttl)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	return &Response{
		ID:           row.ID,
		OriginalName: row.OriginalName,
		ContentType:  row.ContentType,
		SizeBytes:    row.SizeBytes,
		FileURL:      url,
		ExpiresAt:    time.Now().Add(ttl),
	}, nil
}
