package upsertrecord

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/clinical/upsertrecord/db"
	consultationscontracts "tibi/internal/consultations/contracts"
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

type Response struct {
	ID                 int64     `json:"id"`
	ConsultationID     int64     `json:"consultation_id"`
	Allergies          *string   `json:"allergies"`
	CurrentMedications *string   `json:"current_medications"`
	PastConditions     *string   `json:"past_conditions"`
	DoctorNotes        *string   `json:"doctor_notes"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (s *Service) Execute(ctx context.Context, userID, consultationID int64, cmd Command) (*Response, error) {
	cc, err := s.consultations.ClinicalContext(ctx, consultationID, userID)
	if err != nil {
		return nil, err
	}
	if !writableStatus(cc.Status) {
		return nil, httpx.Unprocessable("consultation is not writable: " + cc.Status)
	}

	var resp *Response
	err = s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		existing, err := q.GetRecordByConsultation(ctx, consultationID)
		now := s.clock()

		if err == nil {
			// Update path.
			in := updateInputFrom(cmd, existing)

			// ✅ FIX: تحويل existing.Xmin (string) إلى pgtype.Uint32
			var xmin pgtype.Uint32
			if existing.Xmin != "" {
				val, err := strconv.ParseUint(existing.Xmin, 10, 32)
				if err != nil {
					return httpx.Internal(err)
				}
				xmin = pgtype.Uint32{Uint32: uint32(val), Valid: true}
			}

			n, err := q.UpdateRecord(ctx, db.UpdateRecordParams{
				ID:                 existing.ID,
				Allergies:          in.Allergies,
				CurrentMedications: in.CurrentMedications,
				PastConditions:     in.PastConditions,
				DoctorNotes:        in.DoctorNotes,
				UpdatedAt:          pgtype.Timestamptz{Time: now, Valid: true}, // ✅ استخدام pgtype.Timestamptz مباشرة
				Xmin:               xmin,                                       // ✅ استخدام المتغير المحول
			})
			if err != nil {
				return httpx.Internal(err)
			}
			if n == 0 {
				return httpx.Conflict("record was modified concurrently")
			}
			resp = &Response{
				ID:                 existing.ID,
				ConsultationID:     consultationID,
				Allergies:          in.Allergies,
				CurrentMedications: in.CurrentMedications,
				PastConditions:     in.PastConditions,
				DoctorNotes:        in.DoctorNotes,
				UpdatedAt:          now,
			}
			return nil
		}

		if !errors.Is(err, pgx.ErrNoRows) {
			return httpx.Internal(err)
		}

		// Insert path.
		row, err := q.InsertRecord(ctx, db.InsertRecordParams{
			ConsultationID:     consultationID,
			PatientProfileID:   cc.PatientProfileID,
			DoctorProfileID:    cc.DoctorProfileID,
			Allergies:          cmd.Allergies,
			CurrentMedications: cmd.CurrentMedications,
			PastConditions:     cmd.PastConditions,
			DoctorNotes:        cmd.DoctorNotes,
		})
		if err != nil {
			return httpx.Internal(err)
		}

		resp = &Response{
			ID:                 row.ID,
			ConsultationID:     consultationID,
			Allergies:          cmd.Allergies,
			CurrentMedications: cmd.CurrentMedications,
			PastConditions:     cmd.PastConditions,
			DoctorNotes:        cmd.DoctorNotes,
			UpdatedAt:          row.UpdatedAt.Time,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func writableStatus(s string) bool {
	return s == "InProgress" || s == "Completed"
}

type updateFields struct {
	Allergies          *string
	CurrentMedications *string
	PastConditions     *string
	DoctorNotes        *string
}

func updateInputFrom(cmd Command, existing db.GetRecordByConsultationRow) updateFields {
	pick := func(in *string, cur *string) *string {
		if in != nil {
			return in
		}
		return cur
	}
	return updateFields{
		Allergies:          pick(cmd.Allergies, existing.Allergies),
		CurrentMedications: pick(cmd.CurrentMedications, existing.CurrentMedications),
		PastConditions:     pick(cmd.PastConditions, existing.PastConditions),
		DoctorNotes:        pick(cmd.DoctorNotes, existing.DoctorNotes),
	}
}
