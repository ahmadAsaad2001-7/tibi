package getprescription

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/clinical/getprescription/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type MedicationOut struct {
	ID           int64   `json:"id"`
	Name         string  `json:"medication_name"`
	Dosage       string  `json:"dosage"`
	Frequency    string  `json:"frequency"`
	DurationDays int     `json:"duration_days"`
	Notes        *string `json:"notes,omitempty"`
}

type Response struct {
	ID             int64           `json:"id"`
	ConsultationID int64           `json:"consultation_id"`
	IssuedAt       time.Time       `json:"issued_at"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Medications    []MedicationOut `json:"medications"`
}

func (s *Service) Execute(ctx context.Context, userID, consultationID int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	row, err := q.GetPrescriptionForRead(ctx, db.GetPrescriptionForReadParams{
		ConsultationID: consultationID,
		UserID:         userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("prescription not found")
		}
		return nil, httpx.Internal(err)
	}

	meds, err := q.ListMedications(ctx, row.ID)
	if err != nil {
		return nil, httpx.Internal(err)
	}

	out := make([]MedicationOut, 0, len(meds))
	for _, m := range meds {
		out = append(out, MedicationOut{
			ID:           m.ID,
			Name:         m.MedicationName,
			Dosage:       m.Dosage,
			Frequency:    m.Frequency,
			DurationDays: m.DurationDays,
			Notes:        m.Notes,
		})
	}

	return &Response{
		ID:             row.ID,
		ConsultationID: row.ConsultationID,
		IssuedAt:       row.IssuedAt.Time,
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
		Medications:    out,
	}, nil
}