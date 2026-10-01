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

type InsertParams struct {
	DoctorProfileID int64
	Title           string
	Content         string
	Excerpt         *string
	Type            string
	CoverImageURL   *string
}

type Row struct {
	ID        int64
	CreatedAt time.Time
}

func (q *Queries) InsertPost(ctx context.Context, arg InsertParams) (Row, error) {
	var row Row
	err := q.db.QueryRow(ctx, `
		INSERT INTO content_posts (
		    doctor_profile_id, title, content, excerpt, type, cover_image_url
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		arg.DoctorProfileID, arg.Title, arg.Content, arg.Excerpt, arg.Type, arg.CoverImageURL,
	).Scan(&row.ID, &row.CreatedAt)
	return row, err
}
