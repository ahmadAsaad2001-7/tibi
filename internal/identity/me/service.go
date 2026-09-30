package me

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"tibi/internal/identity/me/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type PatientProfile struct {
	ID                    int64   `json:"id"`
	FullName              string  `json:"full_name"`
	PhoneNumber           string  `json:"phone_number"`
	DateOfBirth           *string `json:"date_of_birth"`
	InsuranceProvider     *string `json:"insurance_provider"`
	InsurancePolicyNumber *string `json:"insurance_policy_number"`
}

type DoctorProfile struct {
	ID                   int64   `json:"id"`
	FullName             string  `json:"full_name"`
	Bio                  string  `json:"bio"`
	ConsultationFee      string  `json:"consultation_fee"`
	Currency             string  `json:"currency"`
	ClinicName           string  `json:"clinic_name"`
	ClinicAddress        string  `json:"clinic_address"`
	MedicalLicenseNumber *string `json:"medical_license_number"`
	IsVerified           bool    `json:"is_verified"`
	VerificationStatus   string  `json:"verification_status"`
	AverageRating        string  `json:"average_rating"`
	RatingCount          int     `json:"rating_count"`
	SpecialtyIDs         []int64 `json:"specialty_ids"`
}

type Response struct {
	ID              int64           `json:"id"`
	Email           string          `json:"email"`
	Role            string          `json:"role"`
	ProfileImageURL *string         `json:"profile_image_url"`
	CreatedAt       string          `json:"created_at"`
	PatientProfile  *PatientProfile `json:"patient_profile"`
	DoctorProfile   *DoctorProfile  `json:"doctor_profile"`
}

func (s *Service) Execute(ctx context.Context, userID int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))

	row, err := q.GetMe(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("user not found")
		}
		return nil, httpx.Internal(err)
	}

	resp := &Response{
		ID:              row.ID,
		Email:           row.Email,
		Role:            string(row.Role),
		ProfileImageURL: row.ProfileImageUrl,
		CreatedAt:       row.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}

	if row.PatientID != nil {
		resp.PatientProfile = &PatientProfile{
			ID:                    *row.PatientID,
			FullName:              derefString(row.PatientFullName),
			PhoneNumber:           derefString(row.PatientPhoneNumber),
			InsuranceProvider:     row.PatientInsuranceProvider,
			InsurancePolicyNumber: row.PatientInsurancePolicyNumber,
		}
		if row.PatientDateOfBirth.Valid {
			s := row.PatientDateOfBirth.Time.Format("2006-01-02")
			resp.PatientProfile.DateOfBirth = &s
		}
	}

	if row.DoctorID != nil {
		specIDs, err := q.GetDoctorSpecialtyIDsForMe(ctx, *row.DoctorID)
		if err != nil {
			return nil, httpx.Internal(err)
		}
		status := derefString(row.DoctorVerificationStatus)
		resp.DoctorProfile = &DoctorProfile{
			ID:                   *row.DoctorID,
			FullName:             derefString(row.DoctorFullName),
			Bio:                  derefString(row.DoctorBio),
			ConsultationFee:      derefString(row.DoctorConsultationFee),
			Currency:             derefString(row.DoctorCurrency),
			ClinicName:           derefString(row.DoctorClinicName),
			ClinicAddress:        derefString(row.DoctorClinicAddress),
			MedicalLicenseNumber: row.DoctorMedicalLicenseNumber,
			IsVerified:           status == "Verified",
			VerificationStatus:   status,
			AverageRating:        derefString(row.DoctorAverageRating),
			RatingCount:          int(derefInt32(row.DoctorRatingCount)),
			SpecialtyIDs:         specIDs,
		}
	}

	return resp, nil
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefInt32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}
