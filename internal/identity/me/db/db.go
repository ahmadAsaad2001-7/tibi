package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/identity/user"
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

const getMe = `-- name: GetMe :one
SELECT
    u.id, u.email, u.role, u.profile_image_url, u.created_at,
    p.id                       AS patient_id,
    p.full_name                AS patient_full_name,
    p.phone_number             AS patient_phone_number,
    p.date_of_birth            AS patient_date_of_birth,
    p.insurance_provider       AS patient_insurance_provider,
    p.insurance_policy_number  AS patient_insurance_policy_number,
    d.id                       AS doctor_id,
    d.full_name                AS doctor_full_name,
    d.bio                      AS doctor_bio,
    d.consultation_fee::text   AS doctor_consultation_fee,
    d.currency                 AS doctor_currency,
    d.clinic_name              AS doctor_clinic_name,
    d.clinic_address           AS doctor_clinic_address,
    d.medical_license_number   AS doctor_medical_license_number,
    d.verification_status      AS doctor_verification_status,
    d.average_rating::text     AS doctor_average_rating,
    d.rating_count             AS doctor_rating_count
FROM identity_users u
LEFT JOIN patients_profiles p ON p.user_id = u.id AND p.deleted_at IS NULL
LEFT JOIN doctors_profiles  d ON d.user_id = u.id AND d.deleted_at IS NULL
WHERE u.id = $1 AND u.deleted_at IS NULL
`

type GetMeRow struct {
	ID                           int64
	Email                        string
	Role                         user.Role
	ProfileImageUrl              *string
	CreatedAt                    pgtype.Timestamptz
	PatientID                    *int64
	PatientFullName              *string
	PatientPhoneNumber           *string
	PatientDateOfBirth           pgtype.Date
	PatientInsuranceProvider     *string
	PatientInsurancePolicyNumber *string
	DoctorID                     *int64
	DoctorFullName               *string
	DoctorBio                    *string
	DoctorConsultationFee        *string
	DoctorCurrency               *string
	DoctorClinicName             *string
	DoctorClinicAddress          *string
	DoctorMedicalLicenseNumber   *string
	DoctorVerificationStatus     *string
	DoctorAverageRating          *string
	DoctorRatingCount            *int32
}

func (q *Queries) GetMe(ctx context.Context, userID int64) (GetMeRow, error) {
	row := q.db.QueryRow(ctx, getMe, userID)
	var i GetMeRow
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.Role,
		&i.ProfileImageUrl,
		&i.CreatedAt,
		&i.PatientID,
		&i.PatientFullName,
		&i.PatientPhoneNumber,
		&i.PatientDateOfBirth,
		&i.PatientInsuranceProvider,
		&i.PatientInsurancePolicyNumber,
		&i.DoctorID,
		&i.DoctorFullName,
		&i.DoctorBio,
		&i.DoctorConsultationFee,
		&i.DoctorCurrency,
		&i.DoctorClinicName,
		&i.DoctorClinicAddress,
		&i.DoctorMedicalLicenseNumber,
		&i.DoctorVerificationStatus,
		&i.DoctorAverageRating,
		&i.DoctorRatingCount,
	)
	return i, err
}

const getDoctorSpecialtyIDsForMe = `-- name: GetDoctorSpecialtyIDsForMe :many
SELECT specialty_id
FROM doctors_profile_specialties
WHERE doctor_profile_id = $1
`

func (q *Queries) GetDoctorSpecialtyIDsForMe(ctx context.Context, doctorProfileID int64) ([]int64, error) {
	rows, err := q.db.Query(ctx, getDoctorSpecialtyIDsForMe, doctorProfileID)
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
