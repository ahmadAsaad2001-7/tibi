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

func (q *Queries) ProfileID(ctx context.Context, userID int64) (int64, error) {
	var id int64
	err := q.db.QueryRow(ctx, `
		SELECT id FROM doctors_profiles WHERE user_id = $1 AND deleted_at IS NULL`, userID).Scan(&id)
	return id, err
}

func (q *Queries) SoftDelete(ctx context.Context, postID, profileID int64) (int64, error) {
	tag, err := q.db.Exec(ctx, `
		UPDATE content_posts
		SET deleted_at = now(), updated_at = now()
		WHERE id = $1 AND doctor_profile_id = $2 AND deleted_at IS NULL`, postID, profileID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
