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

type ConsultationRow struct {
	ID            int64
	Status        string
	ScheduledAt   time.Time
	PatientUserID int64
	DoctorUserID  int64
	Xmin          string
}

func (q *Queries) GetForCancel(ctx context.Context, id int64) (ConsultationRow, error) {
	var row ConsultationRow
	err := q.db.QueryRow(ctx, `
		SELECT c.id, c.status, c.scheduled_at, pp.user_id, dp.user_id, c.xmin::text
		FROM consultations_consultations c
		JOIN patients_profiles pp ON pp.id = c.patient_profile_id
		JOIN doctors_profiles dp ON dp.id = c.doctor_profile_id
		WHERE c.id = $1 AND c.deleted_at IS NULL`, id).Scan(
		&row.ID, &row.Status, &row.ScheduledAt, &row.PatientUserID, &row.DoctorUserID, &row.Xmin,
	)
	return row, err
}

func (q *Queries) UpdateStatus(ctx context.Context, id int64, status string, updatedAt time.Time, xmin string) (int64, error) {
	tag, err := q.db.Exec(ctx, `
		UPDATE consultations_consultations
		SET status = $2, updated_at = $3
		WHERE id = $1 AND xmin::text = $4 AND deleted_at IS NULL`, id, status, updatedAt, xmin)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
