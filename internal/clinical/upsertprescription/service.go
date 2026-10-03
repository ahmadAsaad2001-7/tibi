package upsertprescription

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	consultationscontracts "tibi/internal/consultations/contracts"
	"tibi/internal/clinical/upsertprescription/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db            *database.DB
	consultations consultationscontracts.API
	clock         func() time.Time
}

func NewService(db *database.DB, c consultationscontracts.API) *Service {
	return &Service{db: db, consultations: c, clock: time.Now}
}

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
	UpdatedAt      time.Time       `json:"updated_at"`
	Medications    []MedicationOut `json:"medications"`
}

func (s *Service) Execute(ctx context.Context, userID, consultationID int64, cmd Command) (*Response, error) {
	cc, err := s.consultations.ClinicalContext(ctx, consultationID, userID)
	if err != nil {
		return nil, err
	}
	if cc.Status != "InProgress" && cc.Status != "Completed" {
		return nil, httpx.Unprocessable("consultation is not writable")
	}
	if len(cmd.Medications) == 0 {
		return nil, httpx.Unprocessable("prescription must have at least one medication")
	}

	var resp *Response
	err = s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))
		now := s.clock()

		existing, err := q.GetPrescriptionByConsultation(ctx, consultationID)
		var prescriptionID int64
		var issuedAt, updatedAt time.Time

		if err == nil {
			prescriptionID = existing.ID
			issuedAt = existing.IssuedAt.Time
			updatedAt = now
			if err := q.DeleteMedicationsForPrescription(ctx, prescriptionID); err != nil {
				return httpx.Internal(err)
			}
			if err := q.TouchPrescription(ctx, prescriptionID); err != nil {
				return httpx.Internal(err)
			}
		} else if errors.Is(err, pgx.ErrNoRows) {
			row, err := q.InsertPrescription(ctx, db.InsertPrescriptionParams{
				ConsultationID:   consultationID,
				PatientProfileID: cc.PatientProfileID,
				DoctorProfileID:  cc.DoctorProfileID,
			})
			if err != nil {
				return httpx.Internal(err)
			}
			prescriptionID = row.ID
			issuedAt = row.IssuedAt.Time
			updatedAt = row.UpdatedAt.Time
		} else {
			return httpx.Internal(err)
		}

		meds := make([]MedicationOut, 0, len(cmd.Medications))
		for _, m := range cmd.Medications {
			row, err := q.InsertMedication(ctx, db.InsertMedicationParams{
				PrescriptionID: prescriptionID,
				MedicationName: m.Name,
				Dosage:         m.Dosage,
				Frequency:      m.Frequency,
				DurationDays:   int32(m.DurationDays),
				Notes:          m.Notes,
			})
			if err != nil {
				return httpx.Internal(err)
			}
			meds = append(meds, MedicationOut{
				ID:           row.ID,
				Name:         m.Name,
				Dosage:       m.Dosage,
				Frequency:    m.Frequency,
				DurationDays: m.DurationDays,
				Notes:        m.Notes,
			})
		}

		resp = &Response{
			ID:             prescriptionID,
			ConsultationID: consultationID,
			IssuedAt:       issuedAt,
			UpdatedAt:      updatedAt,
			Medications:    meds,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}