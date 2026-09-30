package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

const insertPatientProfile = `-- name: InsertPatientProfile :one
INSERT INTO patients_profiles (user_id, full_name, phone_number)
VALUES ($1, $2, $3)
    RETURNING id
`

type InsertPatientProfileParams struct {
	UserID      int64
	FullName    string
	PhoneNumber string
}

func (q *Queries) InsertPatientProfile(ctx context.Context, arg InsertPatientProfileParams) (int64, error) {
	row := q.db.QueryRow(ctx, insertPatientProfile, arg.UserID, arg.FullName, arg.PhoneNumber)
	var id int64
	err := row.Scan(&id)
	return id, err
}
