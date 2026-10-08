package replymessage

import (
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// pgTime converts a time.Time to a non-null pgtype.Timestamptz for sqlc.
func pgTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// pgXmin converts an xmin string (from xmin::text) to pgtype.Uint32 for the
// optimistic-lock WHERE clause. Empty/invalid becomes an invalid value.
func pgXmin(x string) pgtype.Uint32 {
	if x == "" {
		return pgtype.Uint32{}
	}
	val, err := strconv.ParseUint(x, 10, 32)
	if err != nil {
		return pgtype.Uint32{}
	}
	return pgtype.Uint32{Uint32: uint32(val), Valid: true}
}
