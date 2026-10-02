package markcompleted

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// pgTime converts a time.Time to a non-null pgtype.Timestamptz for sqlc.
func pgTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
