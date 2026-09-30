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

const insertDoctorProfile = `-- name: InsertDoctorProfile :one
INSERT INTO doctors_profiles (user_id, full_name)
VALUES ($1, $2)
RETURNING id
`

type InsertDoctorProfileParams struct {
	UserID   int64
	FullName string
}

func (q *Queries) InsertDoctorProfile(ctx context.Context, arg InsertDoctorProfileParams) (int64, error) {
	row := q.db.QueryRow(ctx, insertDoctorProfile, arg.UserID, arg.FullName)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const setDoctorVerificationStatus = `-- name: SetDoctorVerificationStatus :execrows
UPDATE doctors_profiles
SET verification_status            = $2,
    verification_rejection_reason  = $3,
    submitted_at                   = CASE WHEN $2::verification_status = 'NotSubmitted'
                                          THEN NULL
                                          ELSE submitted_at END,
    updated_at                     = now()
WHERE user_id = $1 AND deleted_at IS NULL
`

type SetDoctorVerificationStatusParams struct {
	UserID                      int64
	VerificationStatus          string
	VerificationRejectionReason *string
}

func (q *Queries) SetDoctorVerificationStatus(ctx context.Context, arg SetDoctorVerificationStatusParams) (int64, error) {
	tag, err := q.db.Exec(ctx, setDoctorVerificationStatus, arg.UserID, arg.VerificationStatus, arg.VerificationRejectionReason)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
