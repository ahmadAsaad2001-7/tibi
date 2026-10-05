package me

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	filescontracts "tibi/internal/files/contracts"
	"tibi/internal/identity/me/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db    *database.DB
	files filescontracts.API // ✅ Added as per Slice 12b
}

func NewService(db *database.DB, files filescontracts.API) *Service {
	return &Service{db: db, files: files}
}

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

	// ✅ FIX: Resolve Profile Image URL based on Slice 12b spec
	// Note: Because emit_pointers_for_null_types is true for 'me', these are *int64 and *string
	var imageURL *string
	if row.ProfileImageFileID != nil {
		u, err := s.files.PresignGet(ctx, *row.ProfileImageFileID, 86400)
		if err == nil {
			imageURL = &u
		}
	} else if row.ProfileImageUrl != nil {
		imageURL = row.ProfileImageUrl
	}

	resp := &Response{
		ID:              row.ID,
		Email:           row.Email,
		Role:            string(row.Role),
		ProfileImageURL: imageURL, // ✅ Use resolved URL
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

		var status string
		if row.DoctorVerificationStatus.Valid {
			status = string(row.DoctorVerificationStatus.VerificationStatus)
		}

		resp.DoctorProfile = &DoctorProfile{
			ID:                   *row.DoctorID,
			FullName:             derefString(row.DoctorFullName),
			Bio:                  derefString(row.DoctorBio),
			ConsultationFee:      row.DoctorConsultationFee,
			Currency:             derefString(row.DoctorCurrency),
			ClinicName:           derefString(row.DoctorClinicName),
			ClinicAddress:        derefString(row.DoctorClinicAddress),
			MedicalLicenseNumber: row.DoctorMedicalLicenseNumber,
			IsVerified:           status == "Verified",
			VerificationStatus:   status,
			AverageRating:        row.DoctorAverageRating,
			RatingCount:          int(derefInt32(row.DoctorRatingCount)),
			SpecialtyIDs:         specIDs,
		}
	}

	return resp, nil
}

// ✅ Helper functions preserved as requested
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
