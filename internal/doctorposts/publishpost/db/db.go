package db

import (
	"context"
	"time"

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

func (q *Queries) IsPublished(ctx context.Context, postID, profileID int64) (bool, error) {
	var published bool
	err := q.db.QueryRow(ctx, `
		SELECT is_published FROM content_posts
		WHERE id = $1 AND doctor_profile_id = $2 AND deleted_at IS NULL`, postID, profileID).Scan(&published)
	return published, err
}

func (q *Queries) Publish(ctx context.Context, postID, profileID int64, now time.Time) error {
	tag, err := q.db.Exec(ctx, `
		UPDATE content_posts
		SET is_published = true, published_at = $3, updated_at = $3
		WHERE id = $1 AND doctor_profile_id = $2 AND deleted_at IS NULL AND is_published = false`,
		postID, profileID, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
