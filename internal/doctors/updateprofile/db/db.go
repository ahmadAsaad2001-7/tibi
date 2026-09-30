package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// DBTX is the subset of the querier surface used by these queries.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

type Queries struct {
	db DBTX
}

const getDoctorProfileByUserID = `-- name: GetDoctorProfileByUserID :one
SELECT
    id, user_id, full_name, bio,
    consultation_fee::text        AS consultation_fee,
    currency,
    clinic_name, clinic_address, medical_license_number,
    verification_status,
    verification_rejection_reason,
    submitted_at,
    average_rating::text          AS average_rating,
    rating_count,
    created_at, updated_at, deleted_at,
    xmin::text                    AS xmin
FROM doctors_profiles
WHERE user_id = $1 AND deleted_at IS NULL
`

type GetDoctorProfileByUserIDRow struct {
	ID                          int64
	UserID                      int64
	FullName                    string
	Bio                         string
	ConsultationFee             string
	Currency                    string
	ClinicName                  string
	ClinicAddress               string
	MedicalLicenseNumber        *string
	VerificationStatus          string
	VerificationRejectionReason *string
	SubmittedAt                 pgtype.Timestamptz
	AverageRating               string
	RatingCount                 int
	CreatedAt                   pgtype.Timestamptz
	UpdatedAt                   pgtype.Timestamptz
	DeletedAt                   pgtype.Timestamptz
	Xmin                        string
}

func (q *Queries) GetDoctorProfileByUserID(ctx context.Context, userID int64) (GetDoctorProfileByUserIDRow, error) {
	row := q.db.QueryRow(ctx, getDoctorProfileByUserID, userID)
	var i GetDoctorProfileByUserIDRow
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.FullName,
		&i.Bio,
		&i.ConsultationFee,
		&i.Currency,
		&i.ClinicName,
		&i.ClinicAddress,
		&i.MedicalLicenseNumber,
		&i.VerificationStatus,
		&i.VerificationRejectionReason,
		&i.SubmittedAt,
		&i.AverageRating,
		&i.RatingCount,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
		&i.Xmin,
	)
	return i, err
}

const getSpecialtyIDsForDoctor = `-- name: GetSpecialtyIDsForDoctor :many
SELECT specialty_id
FROM doctors_profile_specialties
WHERE doctor_profile_id = $1
`

func (q *Queries) GetSpecialtyIDsForDoctor(ctx context.Context, doctorProfileID int64) ([]int64, error) {
	rows, err := q.db.Query(ctx, getSpecialtyIDsForDoctor, doctorProfileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

const specialtyIDsExist = `-- name: SpecialtyIDsExist :one
SELECT COUNT(*)::bigint AS count
FROM doctors_specialties
WHERE id = ANY($1::bigint[])
`

func (q *Queries) SpecialtyIDsExist(ctx context.Context, specialtyIDs []int64) (int64, error) {
	row := q.db.QueryRow(ctx, specialtyIDsExist, specialtyIDs)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const updateDoctorProfile = `-- name: UpdateDoctorProfile :execrows
UPDATE doctors_profiles
SET
    bio                    = $2,
    consultation_fee       = $3::numeric,
    currency               = $4,
    clinic_name            = $5,
    clinic_address         = $6,
    medical_license_number = $7,
    updated_at             = now()
WHERE id = $1 AND xmin::text = $8
`

type UpdateDoctorProfileParams struct {
	ID                   int64
	Bio                  string
	ConsultationFee      string
	Currency             string
	ClinicName           string
	ClinicAddress        string
	MedicalLicenseNumber *string
	Xmin                 string
}

func (q *Queries) UpdateDoctorProfile(ctx context.Context, arg UpdateDoctorProfileParams) (int64, error) {
	tag, err := q.db.Exec(ctx, updateDoctorProfile,
		arg.ID,
		arg.Bio,
		arg.ConsultationFee,
		arg.Currency,
		arg.ClinicName,
		arg.ClinicAddress,
		arg.MedicalLicenseNumber,
		arg.Xmin,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

const deleteDoctorSpecialties = `-- name: DeleteDoctorSpecialties :exec
DELETE FROM doctors_profile_specialties WHERE doctor_profile_id = $1
`

func (q *Queries) DeleteDoctorSpecialties(ctx context.Context, doctorProfileID int64) error {
	_, err := q.db.Exec(ctx, deleteDoctorSpecialties, doctorProfileID)
	return err
}
