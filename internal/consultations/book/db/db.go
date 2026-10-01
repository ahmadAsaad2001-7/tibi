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

func (q *Queries) GetPatientProfileForUser(ctx context.Context, userID int64) (int64, error) {
	var id int64
	err := q.db.QueryRow(ctx, `
		SELECT id FROM patients_profiles
		WHERE user_id = $1 AND deleted_at IS NULL`, userID).Scan(&id)
	return id, err
}

type CancelStalePendingParams struct {
	DoctorProfileID int64
	ScheduledAt     time.Time
}

func (q *Queries) CancelStalePending(ctx context.Context, arg CancelStalePendingParams) error {
	_, err := q.db.Exec(ctx, `
		UPDATE consultations_consultations
		SET status = 'Cancelled', updated_at = now()
		WHERE doctor_profile_id = $1
		  AND scheduled_at = $2
		  AND status = 'Pending'
		  AND created_at <= now() - INTERVAL '15 minutes'
		  AND deleted_at IS NULL`, arg.DoctorProfileID, arg.ScheduledAt)
	return err
}

type CountActiveSlotParams struct {
	DoctorProfileID int64
	ScheduledAt     time.Time
}

func (q *Queries) CountActiveSlot(ctx context.Context, arg CountActiveSlotParams) (int64, error) {
	var n int64
	err := q.db.QueryRow(ctx, `
		SELECT COUNT(*)::bigint
		FROM consultations_consultations
		WHERE doctor_profile_id = $1
		  AND scheduled_at = $2
		  AND deleted_at IS NULL
		  AND (
		    status IN ('Confirmed', 'InProgress')
		    OR (status = 'Pending' AND created_at > now() - INTERVAL '15 minutes')
		  )`, arg.DoctorProfileID, arg.ScheduledAt).Scan(&n)
	return n, err
}

type InsertConsultationParams struct {
	PatientProfileID int64
	DoctorProfileID  int64
	ClinicSessionID  *int64
	ScheduledAt      time.Time
	DurationMinutes  int32
	IsUrgent         bool
	Notes            *string
}

type ConsultationRow struct {
	ID        int64
	CreatedAt time.Time
}

func (q *Queries) InsertConsultation(ctx context.Context, arg InsertConsultationParams) (ConsultationRow, error) {
	var row ConsultationRow
	err := q.db.QueryRow(ctx, `
		INSERT INTO consultations_consultations (
		    patient_profile_id, doctor_profile_id, clinic_session_id,
		    scheduled_at, duration_minutes, is_urgent, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`,
		arg.PatientProfileID, arg.DoctorProfileID, arg.ClinicSessionID,
		arg.ScheduledAt, arg.DurationMinutes, arg.IsUrgent, arg.Notes,
	).Scan(&row.ID, &row.CreatedAt)
	return row, err
}
