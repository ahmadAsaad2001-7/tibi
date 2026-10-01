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

type PostRow struct {
	ID                    int64
	Title                 string
	Content               string
	Excerpt               string
	Type                  string
	CoverImageURL         *string
	ViewCount             int32
	LikeCount             int32
	PublishedAt           time.Time
	DoctorID              int64
	DoctorFullName        string
	DoctorProfileImageURL *string
}

func (q *Queries) GetAndCountView(ctx context.Context, id int64) (PostRow, error) {
	var row PostRow
	err := q.db.QueryRow(ctx, `
		WITH updated AS (
		    UPDATE content_posts
		    SET view_count = view_count + 1
		    WHERE id = $1 AND is_published AND deleted_at IS NULL
		    RETURNING id, title, content, COALESCE(excerpt, '') AS excerpt, type, cover_image_url,
		              view_count, like_count, COALESCE(published_at, created_at) AS published_at, doctor_profile_id
		)
		SELECT up.id, up.title, up.content, up.excerpt, up.type, up.cover_image_url,
		       up.view_count, up.like_count, up.published_at,
		       d.id, d.full_name, u.profile_image_url
		FROM updated up
		JOIN doctors_profiles d ON d.id = up.doctor_profile_id AND d.deleted_at IS NULL
		JOIN identity_users u ON u.id = d.user_id AND u.deleted_at IS NULL`, id).Scan(
		&row.ID, &row.Title, &row.Content, &row.Excerpt, &row.Type, &row.CoverImageURL,
		&row.ViewCount, &row.LikeCount, &row.PublishedAt,
		&row.DoctorID, &row.DoctorFullName, &row.DoctorProfileImageURL,
	)
	return row, err
}

type SpecialtyRow struct {
	Name string
}

func (q *Queries) Specialties(ctx context.Context, doctorID int64) ([]SpecialtyRow, error) {
	rows, err := q.db.Query(ctx, `
		SELECT s.name
		FROM doctors_profile_specialties dps
		JOIN doctors_specialties s ON s.id = dps.specialty_id
		WHERE dps.doctor_profile_id = $1
		ORDER BY s.name`, doctorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SpecialtyRow, 0)
	for rows.Next() {
		var r SpecialtyRow
		if err := rows.Scan(&r.Name); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
