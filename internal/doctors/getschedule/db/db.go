package db

import (
	"context"

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

type BlockRow struct {
	DayOfWeek           int16
	StartTime           pgtype.Time
	EndTime             pgtype.Time
	SlotDurationMinutes int32
	IsActive            bool
}

func (q *Queries) ListBlocks(ctx context.Context, doctorProfileID int64) ([]BlockRow, error) {
	rows, err := q.db.Query(ctx, `
		SELECT day_of_week, start_time, end_time, slot_duration_minutes, is_active
		FROM doctors_weekly_schedules
		WHERE doctor_profile_id = $1
		ORDER BY day_of_week, start_time`, doctorProfileID)
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
	ID            int64
	ExceptionDate pgtype.Date
	FromTime      pgtype.Time
	ToTime        pgtype.Time
	Type          string
	Reason        *string
}

func (q *Queries) ListExceptions(ctx context.Context, doctorProfileID int64) ([]ExceptionRow, error) {
	rows, err := q.db.Query(ctx, `
		SELECT id, exception_date, from_time, to_time, type, reason
		FROM doctors_schedule_exceptions
		WHERE doctor_profile_id = $1
		ORDER BY exception_date, from_time`, doctorProfileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ExceptionRow, 0)
	for rows.Next() {
		var i ExceptionRow
		if err := rows.Scan(&i.ID, &i.ExceptionDate, &i.FromTime, &i.ToTime, &i.Type, &i.Reason); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
