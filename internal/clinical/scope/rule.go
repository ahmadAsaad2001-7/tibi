package scope

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"tibi/internal/clinical/scope/db"
	filesaccesschecker "tibi/internal/files/accesschecker"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

// Rule authorizes a user to read a file whose scope is MedicalAttachment.
// It succeeds if the user is a participant (patient or doctor) of the
// consultation the file's record belongs to.
type Rule struct{ db *database.DB }

func New(db *database.DB) *Rule { return &Rule{db: db} }

func (r *Rule) AuthorizeFileRead(ctx context.Context, userID int64, fileID int64, uploaderID int64) (bool, error) {
	q := db.New(r.db.Querier(ctx))
	ok, err := q.IsUserParticipantOfFileConsultation(ctx, db.IsUserParticipantOfFileConsultationParams{
		UserID: userID,
		FileID: fileID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, httpx.Internal(err)
	}
	return ok, nil
}

// Satisfy the ScopeRule interface defined by Files.
var _ filesaccesschecker.ScopeRule = (*Rule)(nil)