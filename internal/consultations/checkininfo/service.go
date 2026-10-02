package checkininfo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	consultationscontracts "tibi/internal/consultations/contracts"
	"tibi/internal/consultations/checkininfo/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

func (s *Service) GetForCheckIn(ctx context.Context, consultationID, userID int64) (*consultationscontracts.CheckInInfo, error) {
	q := db.New(s.db.Querier(ctx))
	row, err := q.GetConsultationForCheckIn(ctx, db.GetConsultationForCheckInParams{
		ID:     consultationID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("consultation not found or not confirmable")
		}
		return nil, httpx.Internal(err)
	}
	return &consultationscontracts.CheckInInfo{
		ConsultationID:   row.ID,
		PatientProfileID: row.PatientProfileID,
		DoctorProfileID:  row.DoctorProfileID,
		ClinicSessionID:  row.ClinicSessionID,
		DurationMinutes:  int(row.DurationMinutes),
		IsUrgent:         row.IsUrgent,
	}, nil
}