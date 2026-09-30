package listpendingdoctors

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/admin/listpendingdoctors/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Item struct {
	ID                   int64      `json:"id"`
	UserID               int64      `json:"user_id"`
	Email                string     `json:"email"`
	FullName             string     `json:"full_name"`
	Bio                  string     `json:"bio"`
	ConsultationFee      string     `json:"consultation_fee"`
	Currency             string     `json:"currency"`
	ClinicName           string     `json:"clinic_name"`
	MedicalLicenseNumber *string    `json:"medical_license_number"`
	SubmittedAt          *time.Time `json:"submitted_at"`
}

type Response struct {
	Items []Item `json:"items"`
}

func (s *Service) Execute(ctx context.Context) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	rows, err := q.ListPendingDoctors(ctx)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	items := make([]Item, 0, len(rows))
	for _, r := range rows {
		items = append(items, Item{
			ID:                   r.ID,
			UserID:               r.UserID,
			Email:                r.Email,
			FullName:             r.FullName,
			Bio:                  r.Bio,
			ConsultationFee:      r.ConsultationFee,
			Currency:             r.Currency,
			ClinicName:           r.ClinicName,
			MedicalLicenseNumber: r.MedicalLicenseNumber,
			SubmittedAt:          timePtr(r.SubmittedAt),
		})
	}
	return &Response{Items: items}, nil
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}
