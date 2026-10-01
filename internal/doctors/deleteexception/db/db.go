package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func New(db DBTX) *Queries { return &Queries{db: db} }

type Queries struct{ db DBTX }

func (q *Queries) DeleteException(ctx context.Context, exceptionID, userID int64) (int64, error) {
	tag, err := q.db.Exec(ctx, `
		DELETE FROM doctors_schedule_exceptions e
		USING doctors_profiles p
		WHERE e.id = $1
		  AND e.doctor_profile_id = p.id
		  AND p.user_id = $2
		  AND p.deleted_at IS NULL`, exceptionID, userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
