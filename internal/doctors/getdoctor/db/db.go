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

type DoctorRow struct {
	ID                   int64
	FullName             string
	Bio                  string
	ClinicName           string
	ClinicAddress        string
	ConsultationFee      string
	Currency             string
	AverageRating        string
	RatingCount          int32
	MedicalLicenseNumber *string
	VerificationStatus   string
	ProfileImageURL      *string
}

func (q *Queries) GetDoctorByID(ctx context.Context, id int64) (DoctorRow, error) {
	var row DoctorRow
	err := q.db.QueryRow(ctx, `
		SELECT
		    d.id, d.full_name, d.bio, d.clinic_name, d.clinic_address,
		    d.consultation_fee::text, d.currency,
		    d.average_rating::text, d.rating_count,
		    d.medical_license_number, d.verification_status,
		    u.profile_image_url
		FROM doctors_profiles d
		JOIN identity_users u ON u.id = d.user_id AND u.deleted_at IS NULL
		WHERE d.id = $1
		  AND d.deleted_at IS NULL
		  AND d.verification_status = 'Verified'`, id).Scan(
		&row.ID, &row.FullName, &row.Bio, &row.ClinicName, &row.ClinicAddress,
		&row.ConsultationFee, &row.Currency, &row.AverageRating, &row.RatingCount,
		&row.MedicalLicenseNumber, &row.VerificationStatus, &row.ProfileImageURL,
	)
	return row, err
}

type SpecialtyRow struct {
	ID   int64
	Name string
}

func (q *Queries) SpecialtiesForDoctor(ctx context.Context, id int64) ([]SpecialtyRow, error) {
	rows, err := q.db.Query(ctx, `
		SELECT s.id, s.name
		FROM doctors_profile_specialties dps
		JOIN doctors_specialties s ON s.id = dps.specialty_id
		WHERE dps.doctor_profile_id = $1
		ORDER BY s.name`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SpecialtyRow, 0)
	for rows.Next() {
		var r SpecialtyRow
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type BlockRow struct {
	DayOfWeek           int16
	StartTime           pgtype.Time
	EndTime             pgtype.Time
	SlotDurationMinutes int32
	IsActive            bool
}

func (q *Queries) BlocksForDoctor(ctx context.Context, id int64) ([]BlockRow, error) {
	rows, err := q.db.Query(ctx, `
		SELECT day_of_week, start_time, end_time, slot_duration_minutes, is_active
		FROM doctors_weekly_schedules
		WHERE doctor_profile_id = $1 AND is_active`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]BlockRow, 0)
	for rows.Next() {
		var r BlockRow
		if err := rows.Scan(&r.DayOfWeek, &r.StartTime, &r.EndTime, &r.SlotDurationMinutes, &r.IsActive); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type ExceptionRow struct {
	ExceptionDate time.Time
	FromTime      pgtype.Time
	ToTime        pgtype.Time
	Type          string
}

func (q *Queries) ExceptionsForDoctor(ctx context.Context, id int64, from, to pgtype.Date) ([]ExceptionRow, error) {
	rows, err := q.db.Query(ctx, `
		SELECT exception_date, from_time, to_time, type
		FROM doctors_schedule_exceptions
		WHERE doctor_profile_id = $1
		  AND exception_date BETWEEN $2 AND $3
		  AND type = 'Closed'`, id, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ExceptionRow, 0)
	for rows.Next() {
		var r ExceptionRow
		var day pgtype.Date
		if err := rows.Scan(&day, &r.FromTime, &r.ToTime, &r.Type); err != nil {
			return nil, err
		}
		r.ExceptionDate = day.Time
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Queries) BookedSlots(ctx context.Context, id int64, from, to time.Time) ([]time.Time, error) {
	rows, err := q.db.Query(ctx, `
		SELECT scheduled_at
		FROM consultations_consultations
		WHERE doctor_profile_id = $1
		  AND scheduled_at >= $2 AND scheduled_at < $3
		  AND deleted_at IS NULL
		  AND (
		    status IN ('Confirmed', 'InProgress')
		    OR (status = 'Pending' AND created_at > now() - INTERVAL '15 minutes')
		  )`, id, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]time.Time, 0)
	for rows.Next() {
		var at time.Time
		if err := rows.Scan(&at); err != nil {
			return nil, err
		}
		out = append(out, at)
	}
	return out, rows.Err()
}

type PostRow struct {
	ID          int64
	Title       string
	Excerpt     string
	Type        string
	PublishedAt time.Time
}

func (q *Queries) RecentPosts(ctx context.Context, id int64) ([]PostRow, error) {
	rows, err := q.db.Query(ctx, `
		SELECT id, title, COALESCE(excerpt, ''), type, COALESCE(published_at, created_at)
		FROM content_posts
		WHERE doctor_profile_id = $1
		  AND is_published
		  AND deleted_at IS NULL
		ORDER BY published_at DESC NULLS LAST, id DESC
		LIMIT 5`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PostRow, 0)
	for rows.Next() {
		var r PostRow
		if err := rows.Scan(&r.ID, &r.Title, &r.Excerpt, &r.Type, &r.PublishedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
