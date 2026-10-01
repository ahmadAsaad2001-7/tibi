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

type ListParams struct {
	Type              *string
	Q                 *string
	FeaturedOnly      *bool
	CursorPublishedAt *time.Time
	CursorID          *int64
	LimitCount        int32
}

type PostRow struct {
	ID                    int64
	Title                 string
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

func (q *Queries) ListFeed(ctx context.Context, arg ListParams) ([]PostRow, error) {
	rows, err := q.db.Query(ctx, `
		SELECT p.id, p.title, COALESCE(p.excerpt, ''), p.type, p.cover_image_url,
		       p.view_count, p.like_count, COALESCE(p.published_at, p.created_at),
		       d.id, d.full_name, u.profile_image_url
		FROM content_posts p
		JOIN doctors_profiles d ON d.id = p.doctor_profile_id AND d.deleted_at IS NULL
		JOIN identity_users u ON u.id = d.user_id AND u.deleted_at IS NULL
		WHERE p.is_published AND p.deleted_at IS NULL
		  AND ($1::post_type IS NULL OR p.type = $1::post_type)
		  AND ($2::text IS NULL OR p.title ILIKE '%' || $2 || '%' OR COALESCE(p.excerpt, '') ILIKE '%' || $2 || '%')
		  AND ($3::boolean IS NULL OR $3 = false OR p.is_featured)
		  AND ($4::timestamptz IS NULL OR (COALESCE(p.published_at, p.created_at), p.id) < ($4::timestamptz, $5::bigint))
		ORDER BY COALESCE(p.published_at, p.created_at) DESC, p.id DESC
		LIMIT $6`,
		arg.Type, arg.Q, arg.FeaturedOnly, arg.CursorPublishedAt, arg.CursorID, arg.LimitCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPosts(rows)
}

type SpecialtyRow struct {
	DoctorProfileID int64
	Name            string
}

func (q *Queries) Specialties(ctx context.Context, doctorIDs []int64) ([]SpecialtyRow, error) {
	rows, err := q.db.Query(ctx, `
		SELECT dps.doctor_profile_id, s.name
		FROM doctors_profile_specialties dps
		JOIN doctors_specialties s ON s.id = dps.specialty_id
		WHERE dps.doctor_profile_id = ANY($1::bigint[])
		ORDER BY s.name`, doctorIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SpecialtyRow, 0)
	for rows.Next() {
		var r SpecialtyRow
		if err := rows.Scan(&r.DoctorProfileID, &r.Name); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanPosts(rows pgx.Rows) ([]PostRow, error) {
	out := make([]PostRow, 0)
	for rows.Next() {
		var r PostRow
		if err := rows.Scan(
			&r.ID, &r.Title, &r.Excerpt, &r.Type, &r.CoverImageURL,
			&r.ViewCount, &r.LikeCount, &r.PublishedAt,
			&r.DoctorID, &r.DoctorFullName, &r.DoctorProfileImageURL,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
