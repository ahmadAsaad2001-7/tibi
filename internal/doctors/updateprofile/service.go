package updateprofile

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/doctors/doctorprofile"
	"tibi/internal/doctors/updateprofile/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db *database.DB
}

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Response struct {
	ID                   int64   `json:"id"`
	FullName             string  `json:"full_name"`
	Bio                  string  `json:"bio"`
	ConsultationFee      string  `json:"consultation_fee"`
	Currency             string  `json:"currency"`
	ClinicName           string  `json:"clinic_name"`
	ClinicAddress        string  `json:"clinic_address"`
	MedicalLicenseNumber *string `json:"medical_license_number"`
	VerificationStatus   string  `json:"verification_status"`
	SpecialtyIDs         []int64 `json:"specialty_ids"`
}

func (s *Service) Execute(ctx context.Context, userID int64, cmd Command) (*Response, error) {
	var resp *Response

	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		row, err := q.GetDoctorProfileByUserID(ctx, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("doctor profile not found")
			}
			return httpx.Internal(err)
		}

		// Hydrate aggregate.
		profile := &doctorprofile.Profile{
			ID:                          row.ID,
			UserID:                      row.UserID,
			FullName:                    row.FullName,
			Bio:                         row.Bio,
			ConsultationFee:             row.ConsultationFee,
			Currency:                    row.Currency,
			ClinicName:                  row.ClinicName,
			ClinicAddress:               row.ClinicAddress,
			MedicalLicenseNumber:        row.MedicalLicenseNumber,
			VerificationStatus:          doctorprofile.VerificationStatus(row.VerificationStatus),
			VerificationRejectionReason: row.VerificationRejectionReason,
			SubmittedAt:                 timePtr(row.SubmittedAt),
			AverageRating:               row.AverageRating,
			RatingCount:                 row.RatingCount,
		}

		ids, err := q.GetSpecialtyIDsForDoctor(ctx, row.ID)
		if err != nil {
			return httpx.Internal(err)
		}
		profile.SpecialtyIDs = ids

		// Validate specialty IDs before mutating anything.
		if cmd.SpecialtyIDs != nil {
			unique := dedupe(*cmd.SpecialtyIDs)
			if len(unique) == 0 {
				// Empty array is allowed: clears specialties.
				cmd.SpecialtyIDs = &unique
			} else {
				count, err := q.SpecialtyIDsExist(ctx, unique)
				if err != nil {
					return httpx.Internal(err)
				}
				if count != int64(len(unique)) {
					return httpx.ValidationFailed(map[string]string{
						"specialty_ids": "one or more specialties do not exist",
					})
				}
				cmd.SpecialtyIDs = &unique
			}
		}

		// Apply changes through the domain method.
		profile.UpdateFields(doctorprofile.UpdateInput{
			Bio:                  cmd.Bio,
			ConsultationFee:      cmd.ConsultationFee,
			Currency:             cmd.Currency,
			ClinicName:           cmd.ClinicName,
			ClinicAddress:        cmd.ClinicAddress,
			MedicalLicenseNumber: cmd.MedicalLicenseNumber,
			SpecialtyIDs:         cmd.SpecialtyIDs,
		})

		// Persist with optimistic concurrency.
		affected, err := q.UpdateDoctorProfile(ctx, db.UpdateDoctorProfileParams{
			ID:                   profile.ID,
			Bio:                  profile.Bio,
			ConsultationFee:      profile.ConsultationFee,
			Currency:             profile.Currency,
			ClinicName:           profile.ClinicName,
			ClinicAddress:        profile.ClinicAddress,
			MedicalLicenseNumber: profile.MedicalLicenseNumber,
			Xmin:                 row.Xmin,
		})
		if err != nil {
			return httpx.Internal(err)
		}
		if affected == 0 {
			return httpx.Conflict("profile was modified by another request; retry")
		}

		// Replace specialties if requested.
		if cmd.SpecialtyIDs != nil {
			if err := q.DeleteDoctorSpecialties(ctx, profile.ID); err != nil {
				return httpx.Internal(err)
			}
			if len(*cmd.SpecialtyIDs) > 0 {
				rows := make([][]any, len(*cmd.SpecialtyIDs))
				for i, sid := range *cmd.SpecialtyIDs {
					rows[i] = []any{profile.ID, sid}
				}
				_, err := s.db.Querier(ctx).CopyFrom(ctx,
					pgx.Identifier{"doctors_profile_specialties"},
					[]string{"doctor_profile_id", "specialty_id"},
					pgx.CopyFromRows(rows),
				)
				if err != nil {
					return httpx.Internal(err)
				}
			}
		}

		resp = &Response{
			ID:                   profile.ID,
			FullName:             profile.FullName,
			Bio:                  profile.Bio,
			ConsultationFee:      profile.ConsultationFee,
			Currency:             profile.Currency,
			ClinicName:           profile.ClinicName,
			ClinicAddress:        profile.ClinicAddress,
			MedicalLicenseNumber: profile.MedicalLicenseNumber,
			VerificationStatus:   string(profile.VerificationStatus),
			SpecialtyIDs:         profile.SpecialtyIDs,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return resp, nil
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func dedupe(ids []int64) []int64 {
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
