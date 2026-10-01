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

func (q *Queries) GetVerifiedDoctor(ctx context.Context, doctorProfileID int64) (int64, error) {
	row := q.db.QueryRow(ctx, `
		SELECT id FROM doctors_profiles
		WHERE id = $1 AND verification_status = 'Verified' AND deleted_at IS NULL`, doctorProfileID)
	var id int64
	err := row.Scan(&id)
	return id, err
}

type BlockRow struct {
	DayOfWeek           int16
	StartTime           pgtype.Time
	EndTime             pgtype.Time
	SlotDurationMinutes int32
	IsActive            bool
}

func (q *Queries) BlocksForDoctor(ctx context.Context, doctorProfileID int64) ([]BlockRow, error) {
	rows, err := q.db.Query(ctx, `
		SELECT day_of_week, start_time, end_time, slot_duration_minutes, is_active
		FROM doctors_weekly_schedules
		WHERE doctor_profile_id = $1 AND is_active`, doctorProfileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]BlockRow, 0)
	for rows.Next() {
		var i BlockRow
		if err := rows.Scan(&i.DayOfWeek, &i.StartTime, &i.EndTime, &i.SlotDurationMinutes, &i.IsActive); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

type ExceptionRow struct {
	ExceptionDate time.Time
	FromTime      pgtype.Time
	ToTime        pgtype.Time
	Type          string
}

func (q *Queries) ExceptionsOnDate(ctx context.Context, doctorProfileID int64, date time.Time) ([]ExceptionRow, error) {
	rows, err := q.db.Query(ctx, `
		SELECT exception_date, from_time, to_time, type
		FROM doctors_schedule_exceptions
		WHERE doctor_profile_id = $1 AND exception_date = $2 AND type = 'Closed'`,
		doctorProfileID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ExceptionRow, 0)
	for rows.Next() {
		var i ExceptionRow
		var day pgtype.Date
		if err := rows.Scan(&day, &i.FromTime, &i.ToTime, &i.Type); err != nil {
			return nil, err
		}
		i.ExceptionDate = day.Time
		items = append(items, i)
	}
	return items, rows.Err()
}

func (q *Queries) BookedOnDate(ctx context.Context, doctorProfileID int64, from, to time.Time) ([]time.Time, error) {
	rows, err := q.db.Query(ctx, `
		SELECT scheduled_at
		FROM consultations_consultations
		WHERE doctor_profile_id = $1
		  AND scheduled_at >= $2 AND scheduled_at < $3
		  AND deleted_at IS NULL
		  AND (
		    status IN ('Confirmed', 'InProgress')
		    OR (status = 'Pending' AND created_at > now() - INTERVAL '15 minutes')
		  )`, doctorProfileID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []time.Time
	for rows.Next() {
		var at time.Time
		if err := rows.Scan(&at); err != nil {
			return nil, err
		}
		out = append(out, at)
	}
	return out, rows.Err()
}
