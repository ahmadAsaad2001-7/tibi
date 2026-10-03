package clinicalcontext

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"tibi/internal/consultations/clinicalcontext/db"
	"tibi/internal/consultations/contracts"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

func (s *Service) Get(ctx context.Context, consultationID, userID int64) (*contracts.ClinicalContextInfo, error) {
	q := db.New(s.db.Querier(ctx))
	row, err := q.GetClinicalContext(ctx, db.GetClinicalContextParams{
		ID:     consultationID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("consultation not found")
		}
		return nil, httpx.Internal(err)
	}
	return &contracts.ClinicalContextInfo{
		ConsultationID:   row.ID,
		PatientProfileID: row.PatientProfileID,
		DoctorProfileID:  row.DoctorProfileID,
		Status:           string(row.Status),
	}, nil
}
