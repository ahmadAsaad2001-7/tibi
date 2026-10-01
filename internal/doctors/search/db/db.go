package db

import (
	"context"
	"strings"
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

type SearchBaseParams struct {
	Q               *string
	SpecialtyIds    []int64
	MinFee          *string
	MaxFee          *string
	MinRating       *string
	LimitCount      int32
	CursorRating    *string
	CursorID        *int64
	CursorFee       *string
	CursorCreatedAt *string
}

type SearchRow struct {
	ID              int64
	FullName        string
	ClinicName      string
	Bio             string
	ProfileImageUrl *string
	ConsultationFee string
	Currency        string
	AverageRating   string
	RatingCount     int32
	CreatedAt       time.Time
}

const searchFrom = `
SELECT
    d.id, d.full_name, d.bio, d.clinic_name,
    d.consultation_fee::text, d.currency,
    d.average_rating::text, d.rating_count,
    u.profile_image_url, d.created_at
FROM doctors_profiles d
JOIN identity_users u ON u.id = d.user_id AND u.deleted_at IS NULL
WHERE d.deleted_at IS NULL
  AND d.verification_status = 'Verified'
  AND ($1::text IS NULL
       OR d.full_name ILIKE '%' || $1 || '%'
       OR d.clinic_name ILIKE '%' || $1 || '%'
       OR d.bio ILIKE '%' || $1 || '%')
  AND ($2::bigint[] IS NULL OR EXISTS (
       SELECT 1 FROM doctors_profile_specialties dps
       WHERE dps.doctor_profile_id = d.id
         AND dps.specialty_id = ANY($2::bigint[])
  ))
  AND ($3::numeric IS NULL OR d.consultation_fee >= $3::numeric)
  AND ($4::numeric IS NULL OR d.consultation_fee <= $4::numeric)
  AND ($5::numeric IS NULL OR d.average_rating >= $5::numeric)
`

func (q *Queries) SearchDoctorsByRating(ctx context.Context, arg SearchBaseParams) ([]SearchRow, error) {
	sql := searchFrom + `
  AND ($6::numeric IS NULL OR (d.average_rating, d.id) < ($6::numeric, $7::bigint))
ORDER BY d.average_rating DESC, d.id DESC
LIMIT $8`
	return q.search(ctx, sql, arg.Q, arg.SpecialtyIds, arg.MinFee, arg.MaxFee, arg.MinRating, arg.CursorRating, arg.CursorID, arg.LimitCount)
}

func (q *Queries) SearchDoctorsByFeeAsc(ctx context.Context, arg SearchBaseParams) ([]SearchRow, error) {
	sql := searchFrom + `
  AND ($6::numeric IS NULL
       OR d.consultation_fee > $6::numeric
       OR (d.consultation_fee = $6::numeric AND d.id < $7::bigint))
ORDER BY d.consultation_fee ASC, d.id DESC
LIMIT $8`
	return q.search(ctx, sql, arg.Q, arg.SpecialtyIds, arg.MinFee, arg.MaxFee, arg.MinRating, arg.CursorFee, arg.CursorID, arg.LimitCount)
}

func (q *Queries) SearchDoctorsByFeeDesc(ctx context.Context, arg SearchBaseParams) ([]SearchRow, error) {
	sql := searchFrom + `
  AND ($6::numeric IS NULL OR (d.consultation_fee, d.id) < ($6::numeric, $7::bigint))
ORDER BY d.consultation_fee DESC, d.id DESC
LIMIT $8`
	return q.search(ctx, sql, arg.Q, arg.SpecialtyIds, arg.MinFee, arg.MaxFee, arg.MinRating, arg.CursorFee, arg.CursorID, arg.LimitCount)
}

func (q *Queries) SearchDoctorsByCreatedAt(ctx context.Context, arg SearchBaseParams) ([]SearchRow, error) {
	sql := searchFrom + `
  AND ($6::timestamptz IS NULL OR (d.created_at, d.id) < ($6::timestamptz, $7::bigint))
ORDER BY d.created_at DESC, d.id DESC
LIMIT $8`
	return q.search(ctx, sql, arg.Q, arg.SpecialtyIds, arg.MinFee, arg.MaxFee, arg.MinRating, arg.CursorCreatedAt, arg.CursorID, arg.LimitCount)
}

func (q *Queries) search(ctx context.Context, sql string, args ...any) ([]SearchRow, error) {
	rows, err := q.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SearchRow, 0)
	for rows.Next() {
		var r SearchRow
		if err := rows.Scan(
			&r.ID, &r.FullName, &r.Bio, &r.ClinicName,
			&r.ConsultationFee, &r.Currency, &r.AverageRating, &r.RatingCount,
			&r.ProfileImageUrl, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		r.ConsultationFee = strings.TrimSpace(r.ConsultationFee)
		r.Currency = strings.TrimSpace(r.Currency)
		r.AverageRating = strings.TrimSpace(r.AverageRating)
		out = append(out, r)
	}
	return out, rows.Err()
}

type SpecialtyRow struct {
	DoctorProfileID int64
	SpecialtyID     int64
	Name            string
}

func (q *Queries) SpecialtiesForDoctors(ctx context.Context, doctorIDs []int64) ([]SpecialtyRow, error) {
	rows, err := q.db.Query(ctx, `
		SELECT dps.doctor_profile_id, s.id, s.name
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
		if err := rows.Scan(&r.DoctorProfileID, &r.SpecialtyID, &r.Name); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type BlockRow struct {
	DoctorProfileID     int64
	DayOfWeek           int16
	StartTime           pgtype.Time
	EndTime             pgtype.Time
	SlotDurationMinutes int32
	IsActive            bool
}

func (q *Queries) BlocksForDoctors(ctx context.Context, doctorIDs []int64) ([]BlockRow, error) {
	rows, err := q.db.Query(ctx, `
		SELECT doctor_profile_id, day_of_week, start_time, end_time,
		       slot_duration_minutes, is_active
		FROM doctors_weekly_schedules
		WHERE doctor_profile_id = ANY($1::bigint[])
		  AND is_active`, doctorIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]BlockRow, 0)
	for rows.Next() {
		var r BlockRow
		if err := rows.Scan(&r.DoctorProfileID, &r.DayOfWeek, &r.StartTime, &r.EndTime, &r.SlotDurationMinutes, &r.IsActive); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type ExceptionsForDoctorsParams struct {
	DoctorIds []int64
	FromDate  pgtype.Date
	ToDate    pgtype.Date
}

type ExceptionRow struct {
	DoctorProfileID int64
	ExceptionDate   time.Time
	FromTime        pgtype.Time
	ToTime          pgtype.Time
	Type            string
}

func (q *Queries) ExceptionsForDoctors(ctx context.Context, arg ExceptionsForDoctorsParams) ([]ExceptionRow, error) {
	rows, err := q.db.Query(ctx, `
		SELECT doctor_profile_id, exception_date, from_time, to_time, type
		FROM doctors_schedule_exceptions
		WHERE doctor_profile_id = ANY($1::bigint[])
		  AND exception_date BETWEEN $2 AND $3
		  AND type = 'Closed'`, arg.DoctorIds, arg.FromDate, arg.ToDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ExceptionRow, 0)
	for rows.Next() {
		var r ExceptionRow
		var day pgtype.Date
		if err := rows.Scan(&r.DoctorProfileID, &day, &r.FromTime, &r.ToTime, &r.Type); err != nil {
			return nil, err
		}
		r.ExceptionDate = day.Time
		out = append(out, r)
	}
	return out, rows.Err()
}

type BookedSlotsParams struct {
	DoctorIds []int64
	FromTs    time.Time
	ToTs      time.Time
}

type BookedRow struct {
	DoctorProfileID int64
	ScheduledAt     time.Time
}

func (q *Queries) BookedSlotsForDoctors(ctx context.Context, arg BookedSlotsParams) ([]BookedRow, error) {
	rows, err := q.db.Query(ctx, `
		SELECT doctor_profile_id, scheduled_at
		FROM consultations_consultations
		WHERE doctor_profile_id = ANY($1::bigint[])
		  AND scheduled_at >= $2 AND scheduled_at < $3
		  AND deleted_at IS NULL
		  AND (
		    status IN ('Confirmed', 'InProgress')
		    OR (status = 'Pending' AND created_at > now() - INTERVAL '15 minutes')
		  )`, arg.DoctorIds, arg.FromTs, arg.ToTs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]BookedRow, 0)
	for rows.Next() {
		var r BookedRow
		if err := rows.Scan(&r.DoctorProfileID, &r.ScheduledAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
