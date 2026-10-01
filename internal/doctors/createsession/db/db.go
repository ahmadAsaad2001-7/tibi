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

type DoctorProfile struct {
	ID                 int64
	VerificationStatus string
}

func (q *Queries) GetDoctorProfileByID(ctx context.Context, id int64) (DoctorProfile, error) {
	var row DoctorProfile
	err := q.db.QueryRow(ctx, `
		SELECT id, verification_status
		FROM doctors_profiles
		WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&row.ID, &row.VerificationStatus)
	return row, err
}

type GetDoctorBlockForDateParams struct {
	DoctorProfileID int64
	Date            pgtype.Date
	StartTime       pgtype.Time
	EndTime         pgtype.Time
}

type DoctorBlock struct {
	StartTime           pgtype.Time
	EndTime             pgtype.Time
	SlotDurationMinutes int32
}

func (q *Queries) GetDoctorBlockForDate(ctx context.Context, arg GetDoctorBlockForDateParams) (DoctorBlock, error) {
	var row DoctorBlock
	err := q.db.QueryRow(ctx, `
		SELECT start_time, end_time, slot_duration_minutes
		FROM doctors_weekly_schedules
		WHERE doctor_profile_id = $1
		  AND day_of_week = EXTRACT(DOW FROM $2::date)
		  AND is_active
		  AND start_time <= $3::time
		  AND end_time >= $4::time
		LIMIT 1`,
		arg.DoctorProfileID, arg.Date, arg.StartTime, arg.EndTime).
		Scan(&row.StartTime, &row.EndTime, &row.SlotDurationMinutes)
	return row, err
}

type GetClosedExceptionParams struct {
	DoctorProfileID int64
	ExceptionDate   pgtype.Date
	FromTime        pgtype.Time
	ToTime          pgtype.Time
}

func (q *Queries) GetClosedException(ctx context.Context, arg GetClosedExceptionParams) (int64, error) {
	var id int64
	err := q.db.QueryRow(ctx, `
		SELECT id FROM doctors_schedule_exceptions
		WHERE doctor_profile_id = $1
		  AND exception_date = $2
		  AND from_time <= $3::time
		  AND to_time >= $4::time
		  AND type = 'Closed'
		LIMIT 1`,
		arg.DoctorProfileID, arg.ExceptionDate, arg.FromTime, arg.ToTime).Scan(&id)
	return id, err
}

type GetOrCreateSessionParams struct {
	DoctorProfileID int64
	SessionDate     pgtype.Date
	StartTime       pgtype.Time
	EndTime         pgtype.Time
}

type SessionRow struct {
	ID int64
}

func (q *Queries) GetOrCreateSession(ctx context.Context, arg GetOrCreateSessionParams) (SessionRow, error) {
	var row SessionRow
	err := q.db.QueryRow(ctx, `
		INSERT INTO doctors_clinic_sessions (
		    doctor_profile_id, session_date, start_time, end_time
		) VALUES ($1, $2, $3::time, $4::time)
		ON CONFLICT (doctor_profile_id, session_date, start_time)
		DO UPDATE SET end_time = doctors_clinic_sessions.end_time
		RETURNING id`,
		arg.DoctorProfileID, arg.SessionDate, arg.StartTime, arg.EndTime).Scan(&row.ID)
	return row, err
}
