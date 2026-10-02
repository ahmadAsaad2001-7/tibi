package accesschecker

import (
	"context"

	"tibi/internal/files/getmetadata/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

// Default is the slice-11 access rule. Only the uploader may read their
// own file. Scope-specific rules (consultation participation, public post
// attachments) are added when those consumers adopt the module.
type Default struct{ db *database.DB }

func New(db *database.DB) *Default { return &Default{db: db} }

func (d *Default) Authorize(ctx context.Context, userID, fileID int64) error {
	q := db.New(d.db.Querier(ctx))
	row, err := q.GetFileForAuth(ctx, fileID)
	if err != nil {
		return httpx.NotFound("file not found")
	}
	if row.UploaderID == userID {
		return nil
	}
	// Future: switch on row.Scope and consult the owning module's contract.
	return httpx.NotFound("file not found")
}
