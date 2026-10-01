package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func New(db DBTX) *Queries { return &Queries{db: db} }

type Queries struct{ db DBTX }

func (q *Queries) GetDoctorProfileID(ctx context.Context, userID int64) (int64, error) {
	row := q.db.QueryRow(ctx, `
		SELECT id FROM doctors_profiles
		WHERE user_id = $1 AND deleted_at IS NULL`, userID)
	var id int64
	err := row.Scan(&id)
	return id, err
}

type InsertExceptionParams struct {
	DoctorProfileID int64
	ExceptionDate   time.Time
	FromTime        pgtype.Time
	ToTime          pgtype.Time
	Type            string
	Reason          *string
}

func (q *Queries) InsertException(ctx context.Context, arg InsertExceptionParams) (int64, error) {
	row := q.db.QueryRow(ctx, `
		INSERT INTO doctors_schedule_exceptions (
		    doctor_profile_id, exception_date, from_time, to_time, type, reason
		) VALUES ($1, $2, $3::time, $4::time, $5, $6)
		RETURNING id`,
		arg.DoctorProfileID, arg.ExceptionDate, arg.FromTime, arg.ToTime, arg.Type, arg.Reason)
	var id int64
	err := row.Scan(&id)
	return id, err
}
