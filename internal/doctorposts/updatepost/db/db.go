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
		SELECT id FROM doctors_profiles
		WHERE user_id = $1 AND deleted_at IS NULL`, userID).Scan(&id)
	return id, err
}

type UpdateParams struct {
	ID              int64
	DoctorProfileID int64
	Title           *string
	Content         *string
	Excerpt         *string
	Type            *string
	CoverImageURL   *string
}

type Row struct {
	ID        int64
	UpdatedAt time.Time
}

func (q *Queries) UpdatePost(ctx context.Context, arg UpdateParams) (Row, error) {
	var row Row
	err := q.db.QueryRow(ctx, `
		UPDATE content_posts SET
		    title = COALESCE($3, title),
		    content = COALESCE($4, content),
		    excerpt = COALESCE($5, excerpt),
		    type = COALESCE($6::post_type, type),
		    cover_image_url = COALESCE($7, cover_image_url),
		    updated_at = now()
		WHERE id = $1 AND doctor_profile_id = $2 AND deleted_at IS NULL
		RETURNING id, updated_at`,
		arg.ID, arg.DoctorProfileID, arg.Title, arg.Content, arg.Excerpt, arg.Type, arg.CoverImageURL,
	).Scan(&row.ID, &row.UpdatedAt)
	return row, err
}
