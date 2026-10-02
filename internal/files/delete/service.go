package delete

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"tibi/internal/files/delete/db"
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

// Execute soft-deletes the file record and performs a best-effort deletion
// of the underlying object in storage.
func (s *Service) Execute(ctx context.Context, fileID int64, uploaderID int64) error {
	q := db.New(s.db.Querier(ctx))

	// 1. Soft-delete in the DB and retrieve the object_key
	objectKey, err := q.SoftDeleteFile(ctx, db.SoftDeleteFileParams{
		ID:         fileID,
		UploaderID: uploaderID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.NotFound("file not found or you do not have permission to delete it")
		}
		return httpx.Internal(err)
	}

	_ = s.storage.Delete(ctx, objectKey)

	return nil
}
