package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func New(db DBTX) *Queries { return &Queries{db: db} }

type Queries struct{ db DBTX }

type Row struct {
	ID               int64
	Status           string
	ScheduledAt      time.Time
	DurationMinutes  int32
	IsUrgent         bool
	Notes            *string
	DoctorProfileID  int64
	DoctorName       string
	PatientProfileID int64
	PatientName      string
	PatientUserID    int64
	DoctorUserID     int64
	CreatedAt        time.Time
}

func (q *Queries) GetByID(ctx context.Context, id int64) (Row, error) {
	var row Row
	err := q.db.QueryRow(ctx, `
		SELECT c.id, c.status, c.scheduled_at, c.duration_minutes, c.is_urgent, c.notes,
		       c.doctor_profile_id, d.full_name, c.patient_profile_id, p.full_name,
		       p.user_id, d.user_id, c.created_at
		FROM consultations_consultations c
		JOIN doctors_profiles d ON d.id = c.doctor_profile_id
		JOIN patients_profiles p ON p.id = c.patient_profile_id
		WHERE c.id = $1 AND c.deleted_at IS NULL`, id).Scan(
		&row.ID, &row.Status, &row.ScheduledAt, &row.DurationMinutes, &row.IsUrgent, &row.Notes,
		&row.DoctorProfileID, &row.DoctorName, &row.PatientProfileID, &row.PatientName,
		&row.PatientUserID, &row.DoctorUserID, &row.CreatedAt,
	)
	return row, err
}
