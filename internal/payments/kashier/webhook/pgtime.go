package webhook

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// pgTime converts a time.Time to a non-null pgtype.Timestamptz for sqlc.
func pgTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// tsPtr converts a possibly-null pgtype.Timestamptz to a *time.Time.
func tsPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	tt := t.Time
	return &tt
}
