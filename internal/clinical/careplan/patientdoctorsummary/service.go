package patientdoctorsummary

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/clinical/careplan/patientdoctorsummary/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db *database.DB
}

func NewService(db *database.DB) *Service {
	return &Service{db: db}
}

type Response struct {
	PatientProfileID int64   `json:"patient_profile_id"`
	FullName         string  `json:"full_name"`
	ActiveProtocols  int     `json:"active_protocols"`
	TodayCompleted   int     `json:"today_completed"`
	TodayTotal       int     `json:"today_total"`
	Expected7d       int     `json:"expected_7d"`
	Done7d           int     `json:"done_7d"`
	Adherence7d      float64 `json:"adherence_7d"`
}

func (s *Service) Execute(ctx context.Context, userID, patientID int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))

	// Resolve the doctor profile for the authenticated user.
	doctorID, err := q.GetDoctorProfileIDForUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.Forbidden("doctor profile not found")
		}
		return nil, httpx.Internal(err)
	}

	// The patient must belong to this doctor's care plan.
	owns, err := q.DoctorOwnsPatient(ctx, patientID, doctorID)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	if !owns {
		return nil, httpx.Forbidden("not your patient")
	}

	now := time.Now()
	row, err := q.DoctorPatientSummary(ctx, patientID,
		pgtype.Date{Time: now, Valid: true},
		pgtype.Date{Time: now.AddDate(0, 0, -6), Valid: true},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("patient not found")
		}
		return nil, httpx.Internal(err)
	}

	adh := 0.0
	if row.Expected7d > 0 {
		adh = float64(row.Done7d) / float64(row.Expected7d)
	}

	return &Response{
		PatientProfileID: row.PatientProfileID,
		FullName:         row.FullName,
		ActiveProtocols:  int(row.ActiveProtocols),
		TodayCompleted:   int(row.TodayCompleted),
		TodayTotal:       int(row.TodayTotal),
		Expected7d:       int(row.Expected7d),
		Done7d:           int(row.Done7d),
		Adherence7d:      adh,
	}, nil
}
