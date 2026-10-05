package submitforverification

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/doctors/doctorprofile"
	"tibi/internal/doctors/submitforverification/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db    *database.DB
	clock func() time.Time
}

func NewService(db *database.DB) *Service {
	return &Service{db: db, clock: time.Now}
}

type Response struct {
	ID                 int64      `json:"id"`
	VerificationStatus string     `json:"verification_status"`
	SubmittedAt        *time.Time `json:"submitted_at"`
}

func (s *Service) Execute(ctx context.Context, userID int64) (*Response, error) {
	var resp *Response

	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		row, err := q.GetProfileForSubmission(ctx, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("doctor profile not found")
			}
			return httpx.Internal(err)
		}

		specIDs, err := q.ListSpecialtyIDs(ctx, row.ID)
		if err != nil {
			return httpx.Internal(err)
		}

		// ✅ FIX 1: تحويل pgtype.Text إلى *string بأمان
		var medLicense *string
		if row.MedicalLicenseNumber.Valid {
			medLicense = &row.MedicalLicenseNumber.String
		}

		profile := &doctorprofile.Profile{
			ID:                   row.ID,
			UserID:               row.UserID,
			FullName:             row.FullName,
			Bio:                  row.Bio,
			ConsultationFee:      row.ConsultationFee,
			Currency:             row.Currency,
			ClinicName:           row.ClinicName,
			ClinicAddress:        row.ClinicAddress,
			MedicalLicenseNumber: medLicense, // ✅ تم التصحيح
			VerificationStatus:   doctorprofile.VerificationStatus(row.VerificationStatus),
			SubmittedAt:          timePtr(row.SubmittedAt),
			SpecialtyIDs:         specIDs,
		}

		now := s.clock()
		if err := profile.SubmitForVerification(now); err != nil {
			var incomplete *doctorprofile.IncompleteProfileError
			if errors.As(err, &incomplete) {
				return httpx.Unprocessable("profile incomplete").
					WithDetails(incomplete.Fields())
			}
			if errors.Is(err, doctorprofile.ErrAlreadyVerified) {
				return httpx.Conflict("profile is already verified")
			}
			if errors.Is(err, doctorprofile.ErrAlreadyPendingReview) {
				return httpx.Conflict("profile is already pending review")
			}
			return httpx.Internal(err)
		}

		// ✅ FIX 4: تحويل row.Xmin (string) إلى pgtype.Uint32
		var xmin pgtype.Uint32
		if row.Xmin != "" {
			val, err := strconv.ParseUint(row.Xmin, 10, 32)
			if err == nil {
				xmin = pgtype.Uint32{Uint32: uint32(val), Valid: true}
			}
		}

		// ✅ FIX 2 & 3: استخدام الأنواع الصحيحة لـ UpdateVerificationStatusParams
		affected, err := q.UpdateVerificationStatus(ctx, db.UpdateVerificationStatusParams{
			ID:                 profile.ID,
			VerificationStatus: db.VerificationStatus(doctorprofile.StatusPendingReview), // ✅ تحويل إلى db.VerificationStatus
			SubmittedAt:        pgtype.Timestamptz{Time: now, Valid: true},               // ✅ استخدام pgtype.Timestamptz بدلاً من *time.Time
			Xmin:               xmin,                                                     // ✅ استخدام المتغير المحول
		})
		if err != nil {
			return httpx.Internal(err)
		}
		if affected == 0 {
			return httpx.Conflict("profile was modified by another request; retry")
		}

		resp = &Response{
			ID:                 profile.ID,
			VerificationStatus: string(profile.VerificationStatus),
			SubmittedAt:        profile.SubmittedAt,
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
