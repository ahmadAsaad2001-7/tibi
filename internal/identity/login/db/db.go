package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// DBTX is the subset of the querier surface used by these queries.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

type Queries struct {
	db DBTX
}

const findUserByEmail = `-- name: FindUserByEmail :one
SELECT id, email, password_hash, role, profile_image_url, created_at
FROM identity_users
WHERE email = $1 AND deleted_at IS NULL
`

type UserRow struct {
	ID              int64
	Email           string
	PasswordHash    string
	Role            string
	ProfileImageURL pgtype.Text
	CreatedAt       pgtype.Timestamptz
}

func (q *Queries) FindUserByEmail(ctx context.Context, email string) (UserRow, error) {
	row := q.db.QueryRow(ctx, findUserByEmail, email)
	var i UserRow
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.PasswordHash,
		&i.Role,
		&i.ProfileImageURL,
		&i.CreatedAt,
	)
	return i, err
}