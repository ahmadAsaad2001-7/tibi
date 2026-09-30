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

const listPendingDoctors = `-- name: ListPendingDoctors :many
SELECT
    d.id, d.user_id, d.full_name, d.bio,
    d.consultation_fee::text AS consultation_fee,
    d.currency, d.clinic_name, d.medical_license_number,
    d.submitted_at,
    u.email
FROM doctors_profiles d
         JOIN identity_users u ON u.id = d.user_id
WHERE d.verification_status = 'PendingReview'
  AND d.deleted_at IS NULL
ORDER BY d.submitted_at ASC
`

type ListPendingDoctorsRow struct {
	ID                   int64
	UserID               int64
	FullName             string
	Bio                  string
	ConsultationFee      string
	Currency             string
	ClinicName           string
	MedicalLicenseNumber *string
	SubmittedAt          pgtype.Timestamptz
	Email                string
}

func (q *Queries) ListPendingDoctors(ctx context.Context) ([]ListPendingDoctorsRow, error) {
	rows, err := q.db.Query(ctx, listPendingDoctors)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ListPendingDoctorsRow, 0)
	for rows.Next() {
		var i ListPendingDoctorsRow
		if err := rows.Scan(
			&i.ID, &i.UserID, &i.FullName, &i.Bio,
			&i.ConsultationFee, &i.Currency, &i.ClinicName, &i.MedicalLicenseNumber,
			&i.SubmittedAt, &i.Email,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
