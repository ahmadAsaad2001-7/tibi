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

type Item struct {
	ID               int64
	Status           string
	ScheduledAt      time.Time
	DurationMinutes  int32
	IsUrgent         bool
	DoctorProfileID  int64
	DoctorName       string
	PatientProfileID int64
	PatientName      string
}

func (q *Queries) ListForUser(ctx context.Context, userID int64) ([]Item, error) {
	rows, err := q.db.Query(ctx, `
		SELECT c.id, c.status, c.scheduled_at, c.duration_minutes, c.is_urgent,
		       c.doctor_profile_id, d.full_name, c.patient_profile_id, p.full_name
		FROM consultations_consultations c
		JOIN doctors_profiles d ON d.id = c.doctor_profile_id
		JOIN patients_profiles p ON p.id = c.patient_profile_id
		WHERE c.deleted_at IS NULL
		  AND (p.user_id = $1 OR d.user_id = $1)
		ORDER BY c.scheduled_at DESC
		LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Item, 0)
	for rows.Next() {
		var it Item
		if err := rows.Scan(
			&it.ID, &it.Status, &it.ScheduledAt, &it.DurationMinutes, &it.IsUrgent,
			&it.DoctorProfileID, &it.DoctorName, &it.PatientProfileID, &it.PatientName,
		); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
