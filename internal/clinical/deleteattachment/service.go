package deleteattachment

import (
	"context"

	"tibi/internal/clinical/deleteattachment/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db *database.DB
}

func NewService(db *database.DB) *Service {
	return &Service{db: db}
}

// Execute soft-deletes the attachment.
// It returns a 404-equivalent error if the attachment doesn't exist
// or if the user is not the owning doctor.
func (s *Service) Execute(ctx context.Context, userID, attachmentID int64) error {
	q := db.New(s.db.Querier(ctx))

	// sqlc generates params based on the order of $1, $2 in the query.
	// $1 = a.id (attachmentID), $2 = d.user_id (userID)
	rowsAffected, err := q.SoftDeleteAttachment(ctx, db.SoftDeleteAttachmentParams{
		ID:     attachmentID,
		UserID: userID,
	})
	if err != nil {
		return httpx.Internal(err)
	}

	// Zero rows affected means either the attachment doesn't exist,
	// is already deleted, or the user_id doesn't match the owning doctor.
	if rowsAffected == 0 {
		return httpx.NotFound("attachment not found or you do not have permission to delete it")
	}

	return nil
}