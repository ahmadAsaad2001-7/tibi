package listdoctorpatients

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/clinical/careplan/listdoctorpatients/db"
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

type PatientRow struct {
	PatientProfileID int64   `json:"patient_profile_id"`
	FullName         string  `json:"full_name"`
	ActiveProtocols  int     `json:"active_protocols"`
	TodayCompleted   int     `json:"today_completed"`
	TodayTotal       int     `json:"today_total"`
	Adherence7d      float64 `json:"adherence_7d"`
}

type Response struct {
	Items []PatientRow `json:"items"`
}

func (s *Service) Execute(ctx context.Context, userID int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	docID, err := q.GetDoctorProfileIDForUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.Forbidden("doctor profile not found")
		}
		return nil, httpx.Internal(err)
	}
	patients, err := q.DoctorPatients(ctx, docID)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	today := dateOnly(s.clock())
	from := today.AddDate(0, 0, -6)
	rows := make([]PatientRow, 0, len(patients))
	for _, p := range patients {
		active, err := q.ProtocolsActiveOn(ctx, db.ProtocolsActiveOnParams{
			PatientProfileID: p.PatientProfileID,
			ScheduledDate:    pgtype.Date{Time: today, Valid: true},
		})
		if err != nil {
			return nil, httpx.Internal(err)
		}
		completed := 0
		for _, a := range active {
			if a.MarkedAt.Valid {
				completed++
			}
		}
		var expSum, doneSum int
		for _, a := range active {
			adh, err := q.ProtocolAdherence(ctx, db.ProtocolAdherenceParams{
				ProtocolID: a.ID,
				FromDate:   pgtype.Date{Time: from, Valid: true},
				ToDate:     pgtype.Date{Time: today, Valid: true},
			})
			if err != nil {
				return nil, httpx.Internal(err)
			}
			expSum += int(adh.Expected)
			doneSum += int(adh.Done)
		}
		var adh float64
		if expSum > 0 {
			adh = float64(doneSum) / float64(expSum)
		}
		rows = append(rows, PatientRow{
			PatientProfileID: p.PatientProfileID,
			FullName:         p.FullName,
			ActiveProtocols:  len(active),
			TodayCompleted:   completed,
			TodayTotal:       len(active),
			Adherence7d:      adh,
		})
	}
	return &Response{Items: rows}, nil
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
