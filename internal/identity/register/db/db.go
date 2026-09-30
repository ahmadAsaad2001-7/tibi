package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/identity/user"
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
	Role            user.Role
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

const insertUser = `-- name: InsertUser :one
INSERT INTO identity_users (email, password_hash, role)
VALUES ($1, $2, $3)
RETURNING id, created_at
`

type InsertUserParams struct {
	Email        string
	PasswordHash string
	Role         user.Role
}

type InsertUserRow struct {
	ID        int64
	CreatedAt pgtype.Timestamptz
}

func (q *Queries) InsertUser(ctx context.Context, arg InsertUserParams) (InsertUserRow, error) {
	row := q.db.QueryRow(ctx, insertUser, arg.Email, arg.PasswordHash, arg.Role)
	var i InsertUserRow
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
	)
	return i, err
}

const insertRefreshToken = `-- name: InsertRefreshToken :exec
INSERT INTO identity_refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
`

type InsertRefreshTokenParams struct {
	UserID    int64
	TokenHash string
	ExpiresAt pgtype.Timestamptz
}

func (q *Queries) InsertRefreshToken(ctx context.Context, arg InsertRefreshTokenParams) error {
	_, err := q.db.Exec(ctx, insertRefreshToken, arg.UserID, arg.TokenHash, arg.ExpiresAt)
	return err
}
