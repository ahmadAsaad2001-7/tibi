package call

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// pgTime converts a time.Time to a non-null pgtype.Timestamptz for sqlc.
func pgTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// tsPtr converts a *time.Time to a maybe-null pgtype.Timestamptz for sqlc.
func tsPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// pgToTime converts a maybe-null pgtype.Timestamptz to a *time.Time.
func pgToTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	tt := t.Time
	return &tt
}
